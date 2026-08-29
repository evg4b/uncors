package infra

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientPool(t *testing.T) {
	t.Run("returns a client with a tuned transport", func(t *testing.T) {
		client, err := NewClientPool().For("")

		require.NoError(t, err)
		require.NotNil(t, client)

		transport, ok := client.Transport.(*http.Transport)
		require.True(t, ok)

		assert.Equal(t, maxIdleConnsPerHost, transport.MaxIdleConnsPerHost)
		assert.Equal(t, responseHeaderTimeout, transport.ResponseHeaderTimeout)
		assert.True(t, transport.ForceAttemptHTTP2)

		// A whole-request deadline would cap long downloads and make streamed
		// responses impossible.
		assert.Zero(t, client.Timeout)
	})

	t.Run("reuses one client per upstream proxy setting", func(t *testing.T) {
		pool := NewClientPool()

		first, err := pool.For("")
		require.NoError(t, err)

		second, err := pool.For("")
		require.NoError(t, err)

		viaProxy, err := pool.For("http://localhost:8000")
		require.NoError(t, err)

		assert.Same(t, first, second, "a transport owns a connection pool and must be shared")
		assert.NotSame(t, first, viaProxy)
	})

	t.Run("does not follow redirects", func(t *testing.T) {
		client, err := NewClientPool().For("")
		require.NoError(t, err)

		require.ErrorIs(t, client.CheckRedirect(nil, nil), http.ErrUseLastResponse)
	})

	t.Run("routes through the configured proxy", func(t *testing.T) {
		client, err := NewClientPool().For("http://localhost:8000")
		require.NoError(t, err)

		transport, ok := client.Transport.(*http.Transport)
		require.True(t, ok)
		require.NotNil(t, transport.Proxy)

		proxyURL, err := transport.Proxy(&http.Request{})
		require.NoError(t, err)
		assert.Equal(t, "http://localhost:8000", proxyURL.String())
	})

	t.Run("reports an invalid proxy url instead of panicking", func(t *testing.T) {
		client, err := NewClientPool().For("http://loca^host:8000")

		require.Error(t, err)
		assert.Nil(t, client)
	})

	t.Run("closing releases idle connections", func(t *testing.T) {
		pool := NewClientPool()

		_, err := pool.For("")
		require.NoError(t, err)

		require.NoError(t, pool.Close())
	})
}
