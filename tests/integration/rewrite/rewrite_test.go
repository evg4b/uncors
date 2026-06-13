//go:build integration

package rewrite_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/tests/integration/harness"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func get(t *testing.T, client *http.Client, url string) *http.Response {
	t.Helper()

	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	require.NoError(t, err)

	response, err := client.Do(request)
	require.NoError(t, err)

	return response
}

// requestLine returns the start-line path of the single recorded backend request.
func backendPath(t *testing.T, env *harness.Env) string {
	t.Helper()

	requests := env.Backend.Requests()
	require.Len(t, requests, 1)

	fields := strings.Fields(requests[0]) // "GET /path HTTP/1.1"
	require.GreaterOrEqual(t, len(fields), 2)

	return fields[1]
}

func TestRewriteMiddleware(t *testing.T) {
	env := harness.New(t, harness.WithMapping(func(mapping *config.Mapping) {
		mapping.Rewrites = config.RewriteOptions{
			{From: "/old", To: "/new"},
			{From: "/users/{id}", To: "/accounts/{id}"},
		}
	}))

	t.Run("static path is rewritten before forwarding", func(t *testing.T) {
		env.Backend.Reset()

		response := get(t, env.Client, env.Proxy.HTTPSURL("/old"))
		_ = response.Body.Close()

		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Equal(t, "/new", backendPath(t, env))
	})

	t.Run("placeholder path preserves the captured segment", func(t *testing.T) {
		env.Backend.Reset()

		response := get(t, env.Client, env.Proxy.HTTPSURL("/users/42"))
		_ = response.Body.Close()

		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Equal(t, "/accounts/42", backendPath(t, env))
	})
}
