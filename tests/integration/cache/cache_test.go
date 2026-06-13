//go:build integration

package cache_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/tests/integration/harness"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	cacheMaxSize = 10 * 1024 * 1024
	cacheTTL     = 5 * time.Minute
)

func get(t *testing.T, client *http.Client, url string) {
	t.Helper()

	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	require.NoError(t, err)

	response, err := client.Do(request)
	require.NoError(t, err)

	_ = response.Body.Close()

	require.Equal(t, http.StatusOK, response.StatusCode)
}

func TestCacheMiddleware(t *testing.T) {
	env := harness.New(t,
		harness.WithCacheConfig(config.CacheConfig{
			Methods:        []string{http.MethodGet},
			MaxSize:        cacheMaxSize,
			ExpirationTime: cacheTTL,
		}),
		harness.WithMapping(func(mapping *config.Mapping) {
			mapping.Cache = config.CacheGlobs{"/cached/**"}
		}),
	)

	t.Run("cached path is served from cache after warm-up", func(t *testing.T) {
		env.Backend.Reset()

		url := env.Proxy.HTTPSURL("/cached/resource")

		// Ristretto's Set is asynchronous, so the entry becomes visible shortly
		// after the first miss populates it. Eventually a request is served from
		// cache, leaving the backend count unchanged — which proves a cache hit.
		require.Eventually(t, func() bool {
			before := env.Backend.Count()
			get(t, env.Client, url)

			return env.Backend.Count() == before
		}, 2*time.Second, 10*time.Millisecond)
	})

	t.Run("path outside the cache globs always reaches the backend", func(t *testing.T) {
		env.Backend.Reset()

		const requests = 3
		for range requests {
			get(t, env.Client, env.Proxy.HTTPSURL("/live/resource"))
		}

		assert.Equal(t, requests, env.Backend.Count())
	})
}
