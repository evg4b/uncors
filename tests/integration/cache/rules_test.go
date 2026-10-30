//go:build integration

package cache_test

import (
	"io"
	"net/http"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	cacheWait = 2 * time.Second
	cacheTick = 10 * time.Millisecond
	cacheTTL  = 500 * time.Millisecond
)

func newCacheEnv(t *testing.T) (*integration.Env, *integration.Backend) {
	t.Helper()

	var hits atomic.Int64

	backend := integration.NewBackend(t, func(writer http.ResponseWriter, request *http.Request) {
		hit := strconv.FormatInt(hits.Add(1), 10)

		switch request.URL.Path {
		case "/cached/error":
			writer.WriteHeader(http.StatusInternalServerError)
		case "/cached/cookie":
			http.SetCookie(writer, &http.Cookie{Name: "sid", Value: hit})
		case "/cached/no-store":
			writer.Header().Set("Cache-Control", "no-store")
		case "/cached/private":
			writer.Header().Set("Cache-Control", "private")
		case "/cached/vary-star":
			writer.Header().Set("Vary", "*")
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("X-Upstream", "yes")
		_, err := io.WriteString(writer, `{"hit":`+hit+`}`)
		assert.NoError(t, err)
	})

	env := integration.New(t, backend, &config.UncorsConfig{
		CacheConfig: config.CacheConfig{
			ExpirationTime: cacheTTL,
			MaxSize:        1 << 20,
			Methods:        []string{http.MethodGet},
		},
		Mappings: config.Mappings{{
			From:  hosts.Parse("https://cache.local"),
			To:    backend.AsHost(),
			Cache: config.CacheGlobs{"/cached/**"},
		}},
	})

	return env, backend
}

func get(t *testing.T, env *integration.Env, path string) *integration.Result {
	t.Helper()

	return env.Do(t, integration.NewRequest(t, http.MethodGet, env.RawURL("cache.local", path)))
}

func warm(t *testing.T, env *integration.Env, path string) *integration.Result {
	t.Helper()

	first := get(t, env, path)
	require.True(t, first.HasBackendRequest(), "the first request for %s must be a miss", path)

	require.Eventually(t, func() bool {
		return !get(t, env, path).HasBackendRequest()
	}, cacheWait, cacheTick, "%s was never served from the cache", path)

	return first
}

func TestCacheHit(t *testing.T) {
	env, _ := newCacheEnv(t)

	miss := warm(t, env, "/cached/item")
	hit := get(t, env, "/cached/item")

	t.Run("returns the stored status, headers and body", func(t *testing.T) {
		assert.False(t, hit.HasBackendRequest())
		assert.Equal(t, miss.Response.StatusCode, hit.Response.StatusCode)
		assert.Equal(t, miss.BodyString(), hit.BodyString())
		assert.Equal(t, "application/json", hit.Response.Header.Get("Content-Type"))
		assert.Equal(t, "yes", hit.Response.Header.Get("X-Upstream"))
		assert.Equal(t, "*", hit.Response.Header.Get("Access-Control-Allow-Origin"))
	})

	t.Run("adds an Age header", func(t *testing.T) {
		assert.Empty(t, miss.Response.Header.Get("Age"))
		assert.NotEmpty(t, hit.Response.Header.Get("Age"))
	})

	t.Run("expires after expiration-time", func(t *testing.T) {
		assert.Eventually(t, func() bool {
			return get(t, env, "/cached/item").HasBackendRequest()
		}, cacheWait+cacheTTL, cacheTick)
	})
}

func TestCacheKey(t *testing.T) {
	env, _ := newCacheEnv(t)

	t.Run("sorts query parameter names", func(t *testing.T) {
		warm(t, env, "/cached/query?a=1&b=2")

		assert.False(t, get(t, env, "/cached/query?b=2&a=1").HasBackendRequest())
	})

	t.Run("keeps other queries apart", func(t *testing.T) {
		warm(t, env, "/cached/other?a=1&b=2")

		assert.True(t, get(t, env, "/cached/other?a=2&b=2").HasBackendRequest())
	})

	t.Run("keeps Accept-Encoding variants apart", func(t *testing.T) {
		warm(t, env, "/cached/encoding")

		req := integration.NewRequest(t, http.MethodGet, env.URL("cache.local", "/cached/encoding"))
		req.Header.Set("Accept-Encoding", "br")

		assert.True(t, env.Do(t, req).HasBackendRequest())
	})
}

func TestCacheSkips(t *testing.T) {
	env, backend := newCacheEnv(t)

	paths := map[string]string{
		"a non-2xx response":         "/cached/error",
		"a response with Set-Cookie": "/cached/cookie",
		"Cache-Control: no-store":    "/cached/no-store",
		"Cache-Control: private":     "/cached/private",
		"Vary: *":                    "/cached/vary-star",
	}

	for name, path := range paths {
		t.Run(name+" is not stored", func(t *testing.T) {
			backend.Reset()

			const count = 3
			for range count {
				get(t, env, path)
				time.Sleep(5 * cacheTick)
			}

			assert.Equal(t, count, backend.Count())
		})
	}

	t.Run("a method not in cache-config.methods is not stored", func(t *testing.T) {
		backend.Reset()

		const count = 3
		for range count {
			env.Do(t, integration.NewRequest(t, http.MethodPost, env.URL("cache.local", "/cached/item")))
			time.Sleep(5 * cacheTick)
		}

		assert.Equal(t, count, backend.Count())
	})
}
