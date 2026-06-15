//go:build integration

package cache_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheMiddleware(t *testing.T) {
	backend := integration.NewBackend(t, nil)
	env := integration.New(t, backend, &config.UncorsConfig{
		CacheConfig: config.CacheConfig{
			Methods:        []string{http.MethodGet},
			MaxSize:        10 * 1024 * 1024,
			ExpirationTime: 5 * time.Minute,
		},
		Mappings: config.Mappings{{
			From:  hosts.Parse("https://cache.local"),
			To:    backend.AsHost(),
			Cache: config.CacheGlobs{"/cached/**"},
		}},
	})

	t.Run("cached path is served from cache after warm-up", func(t *testing.T) {
		url := env.URL("cache.local", "/cached/resource")

		// Ristretto's Set is asynchronous, so the entry becomes visible shortly
		// after the first miss. Poll until a request is served from cache
		// (backend count unchanged), proving a cache hit.
		require.Eventually(t, func() bool {
			before := backend.Count()

			req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)

			resp, err := env.Client.Do(req) //nolint:gosec // G704: request targets the in-process test proxy
			if err != nil || resp == nil {
				return false
			}

			resp.Body.Close()

			return backend.Count() == before
		}, 2*time.Second, 10*time.Millisecond)
	})

	t.Run("path outside the cache globs always reaches the backend", func(t *testing.T) {
		backend.Reset()

		const count = 3
		for range count {
			result := env.Do(t, integration.NewRequest(t, http.MethodGet, env.URL("cache.local", "/live/resource")))
			require.Equal(t, http.StatusOK, result.Response.StatusCode)
		}

		assert.Equal(t, count, backend.Count())
	})
}
