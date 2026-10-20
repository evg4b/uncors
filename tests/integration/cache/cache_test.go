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

		// Ristretto's Set is asynchronous.
		require.Eventually(t, func() bool {
			before := backend.Count()

			req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, url, http.NoBody)

			resp, err := env.Client.Do(req)
			if err != nil || resp == nil {
				return false
			}

			resp.Body.Close()

			return backend.Count() == before
		}, 2*time.Second, 10*time.Millisecond)
	})

	t.Run("cache hit answers each origin with its own CORS headers", func(t *testing.T) {
		url := env.URL("cache.local", "/cached/cors")
		request := func(origin string) *http.Request {
			req := integration.NewRequest(t, http.MethodGet, url)
			req.Header.Set("Origin", origin)

			return req
		}

		require.Eventually(t, func() bool {
			return !env.Do(t, request("http://first.local")).HasBackendRequest()
		}, 2*time.Second, 10*time.Millisecond)

		result := env.Do(t, request("http://second.local"))

		assert.False(t, result.HasBackendRequest())
		assert.Equal(t, "ok", result.BodyString())
		assert.Equal(t, "http://second.local", result.Response.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, []string{"Origin"}, result.Response.Header.Values("Vary"))
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
