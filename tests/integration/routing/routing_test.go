//go:build integration

package routing_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/tests/integration/harness"
	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRouting(t *testing.T) {
	// Configure all backend endpoints and proxy logic once, in the parent.
	mux := http.NewServeMux()
	mux.HandleFunc("/api/users", func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(writer, `{"ok":true}`)
	})
	mux.HandleFunc("/ping", func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(writer, "pong")
	})

	env := harness.New(t,
		harness.WithBackendHandler(mux.ServeHTTP),
		harness.WithMapping(func(mapping *config.Mapping) {
			// A route the proxy serves locally; it must never reach the backend.
			mapping.Mocks = config.Mocks{{
				Matcher:  config.RequestMatcher{Path: "/health"},
				Response: config.Response{Code: http.StatusOK, Raw: "healthy"},
			}}
		}),
	)

	get := func(t *testing.T, url string) *http.Response {
		t.Helper()

		request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
		require.NoError(t, err)

		response, err := env.Client.Do(request)
		require.NoError(t, err)

		return response
	}

	t.Run("single request forwarding correctness", func(t *testing.T) {
		env.Backend.Reset()

		response := get(t, env.Proxy.HTTPSURL("/api/users"))
		defer response.Body.Close()

		body, err := io.ReadAll(response.Body)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.JSONEq(t, `{"ok":true}`, string(body))

		// The backend received EXACTLY one request: no duplicate, no retry.
		requests := env.Backend.Requests()
		require.Len(t, requests, 1)
		assert.True(t, strings.HasPrefix(requests[0], "GET /api/users HTTP/1.1"))

		// Snapshot the exact wire-format request the backend received.
		snaps.MatchSnapshot(t, harness.Normalize(requests[0]))
	})

	t.Run("request count validation (no duplicates)", func(t *testing.T) {
		env.Backend.Reset()

		const requestCount = 5
		for range requestCount {
			response := get(t, env.Proxy.HTTPSURL("/ping"))
			require.Equal(t, http.StatusOK, response.StatusCode)
			_ = response.Body.Close()
		}

		// Exactly N hits: proves no duplication, no missing request, no retries.
		assert.Equal(t, requestCount, env.Backend.Count())

		for _, raw := range env.Backend.Requests() {
			assert.True(t, strings.HasPrefix(raw, "GET /ping HTTP/1.1"))
		}
	})

	t.Run("mocked route never reaches backend", func(t *testing.T) {
		env.Backend.Reset()

		response := get(t, env.Proxy.HTTPSURL("/health"))
		defer response.Body.Close()

		body, err := io.ReadAll(response.Body)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Equal(t, "healthy", string(body))

		// The proxy served the mock locally; the backend saw nothing.
		assert.Equal(t, 0, env.Backend.Count())
	})
}
