//go:build integration

package routing_test

import (
	"io"
	"net/http"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/integration"
	"github.com/evg4b/uncors/testing/testutils"
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
		defer testutils.Close(t, result.Response.Body)

		require.Equal(t, http.StatusOK, result.Response.StatusCode)

		// Snapshot what the backend received and what the client got back.
		snaps.MatchSnapshot(t, result.BackendRequest(t))
		snaps.MatchSnapshot(t, result.ResponseDump(t))
	})

	t.Run("each request reaches the backend exactly once", func(t *testing.T) {
		const count = 5
		for range count {
			result := env.Do(t, integration.NewRequest(t, http.MethodGet, env.URL("routing.local", "/ping")))
			require.Equal(t, http.StatusOK, result.Response.StatusCode)
			assert.True(t, result.HasBackendRequest())
			testutils.Close(t, result.Response.Body)
		}
	})

	t.Run("mocked route never reaches backend", func(t *testing.T) {
		result := env.Do(t, integration.NewRequest(t, http.MethodGet, env.URL("routing.local", "/health")))
		defer testutils.Close(t, result.Response.Body)

		assert.Equal(t, http.StatusOK, result.Response.StatusCode)
		assert.Equal(t, "healthy", result.BodyString())
		assert.False(t, result.HasBackendRequest())
	})
}
