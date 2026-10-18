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
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
)

func TestCacheMiddlewareLogs(t *testing.T) {
	serve := func(t *testing.T, status int, err error, times int) []string {
		t.Helper()

		recorded := testutils.Telemetry(t)
		store := lo.Must(cache.NewRistrettoCache(1024*1024, time.Minute))
		middleware := cache.NewMiddleware(
			cache.WithCacheStorage(store),
			cache.WithMethods([]string{http.MethodGet}),
			cache.WithGlobs(config.CacheGlobs{"/api/**"}),
		)
		handler := infra.Wrap(middleware, infra.HandlerFunc(
			func(writer contracts.ResponseWriter, _ *contracts.Request) error {
				writer.WriteHeader(status)

				return err
			},
		))

		testutils.Times(times, func(_ int) {
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://localhost/api/data", nil)
			_ = handler.ServeHTTP(server.NewResponseRecorder(httptest.NewRecorder()), request)

			store.Wait()
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

func TestCacheMiddlewareRejectedStore(t *testing.T) {
	recorded := testutils.Telemetry(t)
	store := lo.Must(cache.NewRistrettoCache(16, time.Minute))
	middleware := cache.NewMiddleware(
		cache.WithCacheStorage(store),
		cache.WithMethods([]string{http.MethodGet}),
		cache.WithGlobs(config.CacheGlobs{"/api/**"}),
	)
	handler := infra.Wrap(middleware, infra.HandlerFunc(
		func(writer contracts.ResponseWriter, _ *contracts.Request) error {
			_, err := writer.Write([]byte("a body larger than the whole cache"))

			return err
		},
	))

	testutils.Times(2, func(_ int) {
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://localhost/api/data", nil)
		require.NoError(t, handler.ServeHTTP(server.NewResponseRecorder(httptest.NewRecorder()), request))
		store.Wait()
	})

	assert.Equal(t, []string{
		"cache miss", "response not cached: the cache storage rejected it",
		"cache miss", "response not cached: the cache storage rejected it",
	}, recorded.Messages())
	assert.Equal(t, int64(2), recorded.Sum(t, "uncors.cache.rejections"))

	for _, span := range recorded.Spans.Ended() {
		if span.Name() == "cache.store" {
			assert.Contains(t, span.Attributes(), attribute.Bool("uncors.cache.rejected", true))
		}
	}
}
