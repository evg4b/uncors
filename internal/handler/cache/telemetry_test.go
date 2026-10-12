package cache_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/cache"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

func TestCacheMiddlewareLogs(t *testing.T) {
	serve := func(t *testing.T, status int, err error, times int) []string {
		t.Helper()

		recorded := testutils.Telemetry(t)
		middleware := cache.NewMiddleware(
			cache.WithCacheStorage(cache.NewRistrettoCache(1024*1024, time.Minute)),
			cache.WithMethods([]string{http.MethodGet}),
			cache.WithGlobs(config.CacheGlobs{"/api/**"}),
		)
		handler := infra.Mddleware(middleware, infra.HandlerFunc(
			func(writer contracts.ResponseWriter, _ *contracts.Request) error {
				writer.WriteHeader(status)

				return err
			},
		))

		testutils.Times(times, func(_ int) {
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://localhost/api/data", nil)
			_ = handler.ServeHTTP(server.NewResponseRecorder(httptest.NewRecorder()), request)
		})

		return recorded.Messages()
	}

	t.Run("stores the response and serves it", func(t *testing.T) {
		assert.Equal(t, []string{"cache miss", "response cached", "cache hit"}, serve(t, http.StatusOK, nil, 2))
	})

	t.Run("skips responses that are not 2xx", func(t *testing.T) {
		assert.Equal(t, []string{"cache miss", "response not cached: the status is not 2xx"},
			serve(t, http.StatusNotFound, nil, 1))
	})

	t.Run("skips requests that failed", func(t *testing.T) {
		assert.Equal(t, []string{"cache miss", "response not cached: the request did not complete"},
			serve(t, http.StatusOK, errUpstreamReset, 1))
	})
}
