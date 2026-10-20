//go:build integration

package routing_test

import (
	"io"
	"net/http"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/integration"
	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRouting(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/users", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"ok":true}`)
		assert.NoError(t, err)
	})
	mux.HandleFunc("/ping", func(w http.ResponseWriter, _ *http.Request) {
		_, err := io.WriteString(w, "pong")
		assert.NoError(t, err)
	})

	backend := integration.NewBackend(t, mux.ServeHTTP)
	env := integration.New(t, backend, &config.UncorsConfig{
		Mappings: config.Mappings{{
			From: hosts.Parse("https://routing.local"),
			To:   backend.AsHost(),
			Mocks: config.Mocks{{
				Matcher:  config.RequestMatcher{Path: "/health"},
				Response: config.Response{Code: http.StatusOK, Raw: "healthy"},
			}},
		}},
	})

	t.Run("forwarded request matches backend and response snapshots", func(t *testing.T) {
		result := env.Do(t, integration.NewRequest(t, http.MethodGet, env.URL("routing.local", "/api/users")))

		require.Equal(t, http.StatusOK, result.Response.StatusCode)

		snaps.MatchSnapshot(t, result.BackendRequest(t))
		snaps.MatchSnapshot(t, result.ResponseDump(t))
	})

	t.Run("each request reaches the backend exactly once", func(t *testing.T) {
		const count = 5
		for range count {
			result := env.Do(t, integration.NewRequest(t, http.MethodGet, env.URL("routing.local", "/ping")))
			require.Equal(t, http.StatusOK, result.Response.StatusCode)
			assert.True(t, result.HasBackendRequest())
		}
	})

	t.Run("mocked route never reaches backend", func(t *testing.T) {
		result := env.Do(t, integration.NewRequest(t, http.MethodGet, env.URL("routing.local", "/health")))

		assert.Equal(t, http.StatusOK, result.Response.StatusCode)
		assert.Equal(t, "healthy", result.BodyString())
		assert.False(t, result.HasBackendRequest())
	})
}

func TestUncleanPaths(t *testing.T) {
	const origin = "https://paths.local"

	backend := integration.NewBackend(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, err := io.WriteString(w, r.Method+" "+r.URL.RequestURI())
		assert.NoError(t, err)
	})
	env := integration.New(t, backend, &config.UncorsConfig{
		Mappings: config.Mappings{{
			From: hosts.Parse(origin),
			To:   backend.AsHost(),
		}},
	})

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{name: "repeated slashes", method: http.MethodPost, path: "//api//v1"},
		{name: "dot segments", method: http.MethodPut, path: "/api/./v1/../v2"},
		{name: "escaped slashes", method: http.MethodPost, path: "/api/a%2F%2Fb"},
		{name: "repeated slashes with query", method: http.MethodGet, path: "/api//v1?page=2"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			req := integration.NewRequest(t, testCase.method, env.RawURL("paths.local", testCase.path))
			req.Header.Set("Origin", origin)

			result := env.Do(t, req)

			assert.Equal(t, http.StatusCreated, result.Response.StatusCode)
			assert.Equal(t, testCase.method+" "+testCase.path, result.BodyString())
			assert.Equal(t, origin, result.Response.Header.Get("Access-Control-Allow-Origin"))
		})
	}
}
