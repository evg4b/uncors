package router_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/go-http-utils/headers"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

const (
	allowedOrigin  = "http://localhost:3000"
	rejectedOrigin = "http://attacker.test"
)

func originsRouter(t *testing.T) http.Handler {
	t.Helper()

	client := mocks.NewHTTPClientMock(t).DoMock.Set(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			Request:    req,
			StatusCode: http.StatusOK,
			Header: http.Header{
				"X-Upstream":                     []string{"1"},
				headers.AccessControlAllowOrigin: []string{"*"},
			},
			Body: io.NopCloser(strings.NewReader("upstream")),
		}, nil
	})

	mapping := func(from string) config.Mapping {
		return config.Mapping{
			From:           hosts.Parse(from),
			To:             hosts.Localhost.HTTPS(),
			AllowedOrigins: config.AllowedOrigins{"http://localhost:*"},
			Mocks:          config.Mocks{mockResponse("/mock", http.StatusCreated, "mock")},
			Scripts: config.Scripts{
				{Matcher: config.RequestMatcher{Path: "/script"}, Script: `response:WriteHeader(202)`},
				{Matcher: config.RequestMatcher{Path: "/fail"}, Script: `error("boom")`},
			},
			Cache: config.CacheGlobs{"/cached/**"},
		}
	}

	forwarded := mapping("http://forwarded.local")
	forwarded.OptionsHandling.Disabled = true

	store := lo.Must(cache.NewRistrettoCache(1<<20, time.Minute))

	t.Cleanup(func() { testutils.Close(t, store) })

	return newRouter(t, testutils.FsFromMap(t, nil), config.Mappings{mapping("http://localhost"), forwarded}, client,
		router.ForRouterWithCacheMiddlewareFactory(func(globs config.CacheGlobs) contracts.Middleware {
			return cache.NewMiddleware(
				cache.WithGlobs(globs),
				cache.WithMethods([]string{http.MethodGet}),
				cache.WithCacheStorage(store),
			)
		}),
	)
}

func serveOrigin(t *testing.T, handler http.Handler, method, url, origin string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(t.Context(), method, url, nil)
	if origin != "" {
		request.Header.Set(headers.Origin, origin)
	}

	helpers.NormaliseRequest(request)

	ctx, scope := tracking.Begin(request.Context(), events.NoopPublisher{}, 1, request, time.Now())
	recorder := httptest.NewRecorder()

	serveHTTP(t, handler, recorder, request.WithContext(ctx))
	scope.Finish(&contracts.RequestData{Method: method, Code: recorder.Code})

	return recorder
}

func corsHeaderNames(header http.Header) []string {
	var names []string

	for name := range header {
		if strings.HasPrefix(name, "Access-Control-") {
			names = append(names, name)
		}
	}

	return names
}

func TestAllowedOrigins(t *testing.T) {
	handler := originsRouter(t)

	routes := []struct {
		name   string
		method string
		url    string
		status int
	}{
		{name: "mock", method: http.MethodGet, url: "http://localhost/mock", status: http.StatusCreated},
		{name: "script", method: http.MethodGet, url: "http://localhost/script", status: http.StatusAccepted},
		{name: "proxy", method: http.MethodGet, url: "http://localhost/api", status: http.StatusOK},
		{name: "error page", method: http.MethodGet, url: "http://localhost/fail", status: http.StatusInternalServerError},
		{name: "forwarded preflight", method: http.MethodOptions, url: "http://forwarded.local/api", status: http.StatusOK},
	}

	for _, route := range routes {
		t.Run(route.name, func(t *testing.T) {
			t.Run("reflects an allowed origin", func(t *testing.T) {
				recorder := serveOrigin(t, handler, route.method, route.url, allowedOrigin)

				assert.Equal(t, route.status, recorder.Code)
				assert.Equal(t, allowedOrigin, recorder.Header().Get(headers.AccessControlAllowOrigin))
				assert.Equal(t, "true", recorder.Header().Get(headers.AccessControlAllowCredentials))
			})

			t.Run("withholds CORS headers from another origin", func(t *testing.T) {
				recorder := serveOrigin(t, handler, route.method, route.url, rejectedOrigin)

				assert.Equal(t, route.status, recorder.Code)
				assert.Empty(t, corsHeaderNames(recorder.Header()))
				assert.Contains(t, recorder.Header().Values(headers.Vary), headers.Origin)
			})

			t.Run("allows a request without an origin", func(t *testing.T) {
				recorder := serveOrigin(t, handler, route.method, route.url, "")

				assert.Equal(t, route.status, recorder.Code)
				assert.Equal(t, "*", recorder.Header().Get(headers.AccessControlAllowOrigin))
			})
		})
	}

	t.Run("preflight", func(t *testing.T) {
		t.Run("is answered for an allowed origin", func(t *testing.T) {
			recorder := serveOrigin(t, handler, http.MethodOptions, "http://localhost/api", allowedOrigin)

			assert.Equal(t, http.StatusOK, recorder.Code)
			assert.Equal(t, allowedOrigin, recorder.Header().Get(headers.AccessControlAllowOrigin))
		})

		t.Run("is forbidden for another origin", func(t *testing.T) {
			recorder := serveOrigin(t, handler, http.MethodOptions, "http://localhost/api", rejectedOrigin)

			assert.Equal(t, http.StatusForbidden, recorder.Code)
			assert.Empty(t, corsHeaderNames(recorder.Header()))
		})

		t.Run("is answered without an origin", func(t *testing.T) {
			recorder := serveOrigin(t, handler, http.MethodOptions, "http://localhost/api", "")

			assert.Equal(t, http.StatusOK, recorder.Code)
			assert.Equal(t, "*", recorder.Header().Get(headers.AccessControlAllowOrigin))
		})
	})

	t.Run("cache hit withholds CORS headers from another origin", func(t *testing.T) {
		miss := serveOrigin(t, handler, http.MethodGet, "http://localhost/cached/item", allowedOrigin)
		assert.Equal(t, allowedOrigin, miss.Header().Get(headers.AccessControlAllowOrigin))

		hit := serveOrigin(t, handler, http.MethodGet, "http://localhost/cached/item", rejectedOrigin)

		assert.Equal(t, http.StatusOK, hit.Code)
		assert.Equal(t, "1", hit.Header().Get("X-Upstream"))
		assert.Empty(t, corsHeaderNames(hit.Header()))
	})
}

func TestAllowedOriginsNotSet(t *testing.T) {
	handler := newRouter(t, testutils.FsFromMap(t, nil), config.Mappings{{
		From:  hosts.Localhost.HTTP(),
		To:    hosts.Localhost.HTTPS(),
		Mocks: config.Mocks{mockResponse("/mock", http.StatusOK, "mock")},
	}}, nil)

	recorder := serveOrigin(t, handler, http.MethodGet, "http://localhost/mock", rejectedOrigin)

	assert.Equal(t, rejectedOrigin, recorder.Header().Get(headers.AccessControlAllowOrigin))
}
