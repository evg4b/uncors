//go:build integration

package events_test

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const closedPort = 1

func newEnv(t *testing.T) *integration.Env {
	t.Helper()

	backend := integration.NewBackend(t, nil)

	return integration.New(t, backend, &config.UncorsConfig{
		CacheConfig: config.CacheConfig{
			Methods:        []string{http.MethodGet},
			MaxSize:        10 * 1024 * 1024,
			ExpirationTime: 5 * time.Minute,
		},
		Mappings: config.Mappings{
			{
				From:    hosts.Parse("https://events.local"),
				To:      backend.AsHost(),
				Statics: config.StaticDirectories{{Path: "/assets", Dir: "/www"}},
				Mocks: config.Mocks{{
					Matcher:  config.RequestMatcher{Path: "/mock"},
					Response: config.Response{Code: http.StatusOK, Raw: "mocked"},
				}},
				Scripts: config.Scripts{{
					Matcher: config.RequestMatcher{Path: "/script"},
					Script:  `response:WriteHeader(201)`,
				}},
				Rewrites: config.RewriteOptions{{From: "/old", To: "/new"}},
				Cache:    config.CacheGlobs{"/cached/**"},
			},
			{
				From: hosts.Parse("https://broken.local"),
				To:   hosts.Loopback.HTTPPort(closedPort),
			},
		},
	}, integration.WithFile("/www/app.js", "console.log('hi')"))
}

func TestRequestAttribution(t *testing.T) {
	env := newEnv(t)

	tests := []struct {
		name    string
		method  string
		path    string
		handler events.HandlerKind
		code    int
	}{
		{name: "mock", method: http.MethodGet, path: "/mock", handler: events.HandlerMock, code: http.StatusOK},
		{name: "script", method: http.MethodGet, path: "/script", handler: events.HandlerScript, code: http.StatusCreated},
		{
			name:    "static file",
			method:  http.MethodGet,
			path:    "/assets/app.js",
			handler: events.HandlerStatic,
			code:    http.StatusOK,
		},
		{
			name:    "static fall-through",
			method:  http.MethodGet,
			path:    "/assets/missing.js",
			handler: events.HandlerProxy,
			code:    http.StatusOK,
		},
		{name: "preflight", method: http.MethodOptions, path: "/api", handler: events.HandlerOptions, code: http.StatusOK},
		{name: "proxied request", method: http.MethodGet, path: "/api", handler: events.HandlerProxy, code: http.StatusOK},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			request := integration.NewRequest(t, testCase.method, env.URL("events.local", testCase.path))

			result, tracked := env.DoTracked(t, request)

			assert.Equal(t, testCase.code, result.Response.StatusCode)
			assert.Equal(t, tracked.Started.ID, tracked.Completed.ID)
			assert.Equal(t, testCase.handler, tracked.Started.Handler, "started")
			assert.Equal(t, testCase.handler, tracked.Completed.Handler, "completed")
			assert.Equal(t, testCase.code, tracked.Completed.Data.Code)
			require.NoError(t, tracked.Completed.Err)
			assert.Nil(t, tracked.Completed.Rewrite)
		})
	}

	t.Run("a cache miss is proxied and a repeat is served from the cache", func(t *testing.T) {
		url := env.URL("events.local", "/cached/item")

		_, first := env.DoTracked(t, integration.NewRequest(t, http.MethodGet, url))
		assert.Equal(t, events.HandlerProxy, first.Completed.Handler)

		// The response is stored once it has been sent, so the very next
		// request can still miss.
		assert.Eventually(t, func() bool {
			_, next := env.DoTracked(t, integration.NewRequest(t, http.MethodGet, url))

			return next.Started.Handler == events.HandlerCache && next.Completed.Handler == events.HandlerCache
		}, 2*time.Second, 10*time.Millisecond)
	})

	t.Run("a rewrite is proxied and reports the new path", func(t *testing.T) {
		_, tracked := env.DoTracked(t, integration.NewRequest(t, http.MethodGet, env.URL("events.local", "/old")))

		rewrite := &events.Rewrite{From: "/old", To: "/new"}

		assert.Equal(t, events.HandlerProxy, tracked.Completed.Handler)
		assert.Equal(t, rewrite, tracked.Started.Rewrite)
		assert.Equal(t, rewrite, tracked.Completed.Rewrite)
	})

	t.Run("an unreachable upstream reports the proxy error", func(t *testing.T) {
		result, tracked := env.DoTracked(t, integration.NewRequest(t, http.MethodGet, env.URL("broken.local", "/api")))

		assert.Equal(t, http.StatusBadGateway, result.Response.StatusCode)
		assert.Equal(t, events.HandlerProxy, tracked.Completed.Handler)
		require.Error(t, tracked.Completed.Err)
		assert.Contains(t, tracked.Completed.Err.Error(), "connection refused")
	})

	t.Run("a host no mapping covers is not attributed", func(t *testing.T) {
		env.Hosts.Set("unmapped.local", "127.0.0.1")
		port := strconv.Itoa(env.PortFor("events.local"))
		url := "https://unmapped.local:" + port + "/api"

		result, tracked := env.DoTracked(t, integration.NewRequest(t, http.MethodGet, url))

		assert.Equal(t, http.StatusNotFound, result.Response.StatusCode)
		assert.Equal(t, events.HandlerNone, tracked.Started.Handler)
		assert.Equal(t, events.HandlerNone, tracked.Completed.Handler)
		require.Error(t, tracked.Completed.Err)
		assert.Contains(t, tracked.Completed.Err.Error(), "host not mapped")
	})
}

func TestEngineAnnouncesStart(t *testing.T) {
	env := newEnv(t)

	received := env.Events()
	require.NotEmpty(t, received)

	started, ok := received[0].(events.ServerStarted)
	require.True(t, ok, "the first event is %T", received[0])
	assert.Contains(t, started.Mappings.String(), "events.local")
	assert.Contains(t, started.Mappings.String(), "broken.local")
}
