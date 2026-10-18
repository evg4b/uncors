//go:build integration

package proxy_test

import (
	"net/http"
	"testing"

	"github.com/evg4b/uncors/testing/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProxyHopByHopHeaders(t *testing.T) {
	env, _ := newProxyEnv(t)

	t.Run("does not pass hop-by-hop response headers to the client", func(t *testing.T) {
		result := env.Do(t, integration.NewRequest(t, http.MethodGet, proxyURL(env, "/hop")))

		require.Equal(t, http.StatusOK, result.Response.StatusCode)
		assert.Equal(t, "hop", result.BodyString())

		for _, name := range []string{"Connection", "X-Hop", "Keep-Alive"} {
			assert.Empty(t, result.Response.Header.Values(name), name)
		}
	})

	t.Run("passes Content-Length of a HEAD response through", func(t *testing.T) {
		result := env.Do(t, integration.NewRequest(t, http.MethodHead, proxyURL(env, "/sized")))

		require.Equal(t, http.StatusOK, result.Response.StatusCode)
		assert.Equal(t, int64(1234), result.Response.ContentLength)
	})
}
