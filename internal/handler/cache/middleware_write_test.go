package cache_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/cache"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errClientGone = errors.New("client gone")

type disconnectedWriter struct {
	*httptest.ResponseRecorder
}

func (disconnectedWriter) Write([]byte) (int, error) {
	return 0, errClientGone
}

func TestCacheMiddlewareClientDisconnect(t *testing.T) {
	store := lo.Must(cache.NewRistrettoCache(1024*1024, time.Minute))
	middleware := cache.NewMiddleware(
		cache.WithCacheStorage(store),
		cache.WithMethods([]string{http.MethodGet}),
		cache.WithGlobs(config.CacheGlobs{"/api/**"}),
	)

	handler := testutils.NewCounter(func(writer contracts.ResponseWriter, _ *contracts.Request) error {
		writer.WriteHeader(http.StatusOK)
		fmt.Fprint(writer, "cached body")

		return nil
	})

	serve := func(writer http.ResponseWriter) error {
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/data", nil)

		return middleware.ServeHTTP(server.NewResponseRecorder(writer), request, handler.ServeHTTP)
	}

	require.NoError(t, serve(httptest.NewRecorder()))
	store.Wait()

	recorder := httptest.NewRecorder()
	require.NoError(t, serve(recorder))
	assert.Equal(t, "cached body", testutils.ReadBody(t, recorder))
	assert.Equal(t, 1, handler.Count())

	recorder = httptest.NewRecorder()

	assert.NotPanics(t, func() {
		require.NoError(t, serve(disconnectedWriter{recorder}))
	})
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, 1, handler.Count())
}
