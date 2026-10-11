//go:build integration

package rewrite_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/integration"
	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func backendPath(t *testing.T, result *integration.Result) string {
	t.Helper()

	raw := result.BackendRequest(t)
	fields := strings.Fields(raw)
	require.GreaterOrEqual(t, len(fields), 2)

	return fields[1]
}

func TestRewriteMiddleware(t *testing.T) {
	backend := integration.NewBackend(t, nil)
	env := integration.New(t, backend, &config.UncorsConfig{
		Mappings: config.Mappings{{
			From: hosts.Parse("https://rewrite.local"),
			To:   backend.AsHost(),
			Rewrites: config.RewriteOptions{
				{From: "/old", To: "/new"},
				{From: "/users/{id}", To: "/accounts/{id}"},
			},
		}},
	})

	t.Run("static path is rewritten before forwarding", func(t *testing.T) {
		result := env.Do(t, integration.NewRequest(t, http.MethodGet, env.URL("rewrite.local", "/old")))

		assert.Equal(t, http.StatusOK, result.Response.StatusCode)
		assert.Equal(t, "/new", backendPath(t, result))

		snaps.MatchSnapshot(t, result.BackendRequest(t))
		snaps.MatchSnapshot(t, result.ResponseDump(t))
	})

	t.Run("placeholder path preserves the captured segment", func(t *testing.T) {
		result := env.Do(t, integration.NewRequest(t, http.MethodGet, env.URL("rewrite.local", "/users/42")))

		assert.Equal(t, http.StatusOK, result.Response.StatusCode)
		assert.Equal(t, "/accounts/42", backendPath(t, result))

		snaps.MatchSnapshot(t, result.BackendRequest(t))
		snaps.MatchSnapshot(t, result.ResponseDump(t))
	})
}
