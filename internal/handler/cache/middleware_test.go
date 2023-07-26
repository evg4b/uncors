package cache_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/cache"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/go-http-utils/headers"
	goCache "github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
)

func TestCacheMiddleware(t *testing.T) {
	const expectedBody = "this is test"

	expectedHeader := http.Header{
		headers.ContentType:     {"text/html; charset=iso-8859-1"},
		headers.ContentEncoding: {"deflate, gzip"},
	}

	middleware := cache.NewMiddleware(
		cache.WithCacheStorage(goCache.New(time.Minute, time.Minute)),
		cache.WithLogger(mocks.NewNoopLogger(t)),
		cache.WithMethods([]string{http.MethodGet}),
		cache.WithGlobs(config.CacheGlobs{
			"/translations",
			"/api/**",
		}),
	)

	handler := testutils.NewCounter(func(writer contracts.ResponseWriter, request *contracts.Request) {
		writer.WriteHeader(http.StatusOK)
		testutils.CopyHeaders(expectedHeader, writer.Header())
		helpers.Fprintf(writer, expectedBody)
	})

	t.Run("should not call cached response just one time for", func(t *testing.T) {
		tests := []struct {
			name       string
			method     string
			path       string
			statusCode int
		}{
			{
				name:       "request in glob",
				method:     http.MethodGet,
				path:       "/api",
				statusCode: http.StatusOK,
			},
			{
				name:       "request in glob with params",
				method:     http.MethodGet,
				path:       "/api?some=params",
				statusCode: http.StatusOK,
			},
			{
				name:       "request in glob with other params",
				method:     http.MethodGet,
				path:       "/api?other=params",
				statusCode: http.StatusOK,
			},
			{
				name:       "second level request in glob",
				method:     http.MethodGet,
				path:       "/api/comments",
				statusCode: http.StatusOK,
			},
			{
				name:       "second level request in glob with params",
				method:     http.MethodGet,
				path:       "/api/comments?q=test",
				statusCode: http.StatusOK,
			},
			{
				name:       "third level request in glob",
				method:     http.MethodGet,
				path:       "/api/comments/1",
				statusCode: http.StatusOK,
			},
			{
				name:       "third level request in glob with params",
				method:     http.MethodGet,
				path:       "/api/comments/1?q=demo",
				statusCode: http.StatusOK,
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				handler.Reset()

				wrappedHandler := middleware.Wrap(handler)

				testutils.Times(5, func(_ int) {
					recorder := httptest.NewRecorder()
					wrappedHandler.ServeHTTP(
						contracts.WrapResponseWriter(recorder),
						httptest.NewRequest(testCase.method, testCase.path, nil),
					)
					assert.Equal(t, expectedHeader, recorder.Header())
					assert.Equal(t, expectedBody, testutils.ReadBody(t, recorder))
				})

				assert.Equal(t, 1, handler.Count())
			})
		}
	})

	t.Run("should not cache response", func(t *testing.T) {
		tests := []struct {
			name       string
			method     string
			path       string
			statusCode int
		}{
			{
				name:       "witch path out of glob",
				method:     http.MethodGet,
				path:       "/test",
				statusCode: http.StatusOK,
			},
			{
				name:       "from POST method request",
				method:     http.MethodPost,
				path:       "/api",
				statusCode: http.StatusOK,
			},
			{
				name:       "witch response with status code 500",
				method:     http.MethodGet,
				path:       "/api/constants",
				statusCode: http.StatusInternalServerError,
			},
			{
				name:       "witch response with status code 400",
				method:     http.MethodGet,
				path:       "/api/constants",
				statusCode: http.StatusBadRequest,
			},
			{
				name:       "witch response with status code 304",
				method:     http.MethodGet,
				path:       "/api/constants",
				statusCode: http.StatusNotModified,
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				handler := testutils.NewCounter(func(writer contracts.ResponseWriter, request *contracts.Request) {
					writer.WriteHeader(testCase.statusCode)
					testutils.CopyHeaders(expectedHeader, writer.Header())
					helpers.Fprintf(writer, expectedBody)
				})

				wrappedHandler := middleware.Wrap(handler)

				testutils.Times(5, func(_ int) {
					recorder := httptest.NewRecorder()
					wrappedHandler.ServeHTTP(
						contracts.WrapResponseWriter(recorder),
						httptest.NewRequest(testCase.method, testCase.path, nil),
					)
					assert.Equal(t, expectedHeader, recorder.Header())
					assert.Equal(t, expectedBody, testutils.ReadBody(t, recorder))
				})

				assert.Equal(t, 5, handler.Count())
			})
		}
	})

	t.Run("should not cache response between different hosts matched by one rule", func(t *testing.T) {
		const count = 5
		handler.Reset()

		middleware := cache.NewMiddleware(
			cache.WithCacheStorage(goCache.New(time.Minute, time.Minute)),
			cache.WithLogger(mocks.NewNoopLogger(t)),
			cache.WithMethods([]string{http.MethodGet}),
			cache.WithGlobs(config.CacheGlobs{"/api/**"}),
		)

		wrappedHandler := middleware.Wrap(handler)

		testutils.Times(count, func(index int) {
			recorder := httptest.NewRecorder()
			url := fmt.Sprintf("https://test-host-%d.com:4200/api/test", index)
			request := httptest.NewRequest(http.MethodGet, url, nil)
			wrappedHandler.ServeHTTP(
				contracts.WrapResponseWriter(recorder),
				request,
			)
			assert.Equal(t, expectedHeader, recorder.Header())
			assert.Equal(t, expectedBody, testutils.ReadBody(t, recorder))
		})

		assert.Equal(t, count, handler.Count())
	})
}
