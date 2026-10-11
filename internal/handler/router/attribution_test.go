package router_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/handler/cache"
	"github.com/evg4b/uncors/internal/handler/router"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/tracking"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func attributionRouter(t *testing.T) http.Handler {
	t.Helper()

	fs := testutils.FsFromMap(t, map[string]string{
		"/assets/app.js": "app",
	})

	mappings := config.Mappings{
		{
			From:    hosts.Localhost.HTTP(),
			To:      hosts.Localhost.HTTPS(),
			Statics: config.StaticDirectories{{Dir: "/assets", Path: "/static/"}},
			Mocks: config.Mocks{{
				Matcher:  config.RequestMatcher{Path: "/mock"},
				Response: config.Response{Code: http.StatusOK, Raw: "mock"},
			}},
			Scripts: config.Scripts{{
				Matcher: config.RequestMatcher{Path: "/script"},
				Script:  `response:WriteHeader(200)`,
			}},
			Rewrites: config.RewriteOptions{{From: "/old", To: "/new"}},
			Cache:    config.CacheGlobs{"/cached/**"},
		},
	}

	store := cache.NewRistrettoCache(1<<20, time.Minute)

	t.Cleanup(func() { testutils.Close(t, store) })

	return newRouter(t, fs, mappings, upstream(t, http.StatusOK, "upstream"),
		router.ForRouterWithCacheMiddlewareFactory(func(globs config.CacheGlobs) contracts.Middleware {
			return cache.NewMiddleware(
				cache.WithGlobs(globs),
				cache.WithMethods([]string{http.MethodGet}),
				cache.WithCacheStorage(store),
			)
		}),
	)
}

func serveTracked(
	t *testing.T,
	handler http.Handler,
	method, url string,
) (events.RequestStarted, events.RequestCompleted) {
	t.Helper()

	bus := events.NewBus()
	subscription := bus.Subscribe(2)

	request := httptest.NewRequestWithContext(t.Context(), method, url, nil)
	helpers.NormaliseRequest(request)

	ctx, scope := tracking.Begin(request.Context(), bus, 1, request)
	recorder := httptest.NewRecorder()

	serveHTTP(t, handler, recorder, request.WithContext(ctx))
	scope.Finish(helpers.ToRequestData(request, recorder.Code))
	bus.Close()

	var received []events.Event
	for event := range subscription.Events() {
		received = append(received, event)
	}

	require.Len(t, received, 2)

	started, ok := received[0].(events.RequestStarted)
	require.True(t, ok)

	completed, ok := received[1].(events.RequestCompleted)
	require.True(t, ok)

	return started, completed
}

func TestRouterAttribution(t *testing.T) {
	handler := attributionRouter(t)

	tests := []struct {
		name    string
		method  string
		url     string
		handler events.HandlerKind
	}{
		{name: "mock", method: http.MethodGet, url: "http://localhost/mock", handler: events.HandlerMock},
		{name: "script", method: http.MethodGet, url: "http://localhost/script", handler: events.HandlerScript},
		{name: "static file", method: http.MethodGet, url: "http://localhost/static/app.js", handler: events.HandlerStatic},
		{
			name:    "static fall-through",
			method:  http.MethodGet,
			url:     "http://localhost/static/missing.js",
			handler: events.HandlerProxy,
		},
		{name: "preflight", method: http.MethodOptions, url: "http://localhost/api", handler: events.HandlerOptions},
		{name: "proxied request", method: http.MethodGet, url: "http://localhost/api", handler: events.HandlerProxy},
		{name: "unmapped host", method: http.MethodGet, url: "http://unknown.local/api", handler: events.HandlerNone},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			started, completed := serveTracked(t, handler, testCase.method, testCase.url)

			assert.Equal(t, testCase.handler, started.Handler)
			assert.Equal(t, testCase.handler, completed.Handler)
			assert.Nil(t, completed.Rewrite)
		})
	}

	t.Run("cache miss is proxied, the next request is a cache hit", func(t *testing.T) {
		started, completed := serveTracked(t, handler, http.MethodGet, "http://localhost/cached/item")
		assert.Equal(t, events.HandlerProxy, started.Handler)
		assert.Equal(t, events.HandlerProxy, completed.Handler)

		started, completed = serveTracked(t, handler, http.MethodGet, "http://localhost/cached/item")
		assert.Equal(t, events.HandlerCache, started.Handler)
		assert.Equal(t, events.HandlerCache, completed.Handler)
	})

	t.Run("rewrite is proxied and reports the path change", func(t *testing.T) {
		started, completed := serveTracked(t, handler, http.MethodGet, "http://localhost/old")

		rewrite := &events.Rewrite{From: "/old", To: "/new"}

		assert.Equal(t, events.HandlerProxy, started.Handler)
		assert.Equal(t, rewrite, started.Rewrite)
		assert.Equal(t, events.HandlerProxy, completed.Handler)
		assert.Equal(t, rewrite, completed.Rewrite)
	})

	t.Run("unmapped host reports the routing error", func(t *testing.T) {
		_, completed := serveTracked(t, handler, http.MethodGet, "http://unknown.local/api")

		require.Error(t, completed.Err)
		assert.Contains(t, completed.Err.Error(), "host not mapped")
	})
}
