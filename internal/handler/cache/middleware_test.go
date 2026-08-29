package cache_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/handler/cache"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/go-http-utils/headers"
	"github.com/stretchr/testify/assert"
)

// methodBody keeps the response body a value derived from a known set rather
// than an echo of request data.
func methodBody(method string) string {
	return "handled " + method
}

func TestCacheMiddleware(t *testing.T) {
	const (
		expectedBody     = "this is test"
		cacheGlob        = "/api/**"
		constantEndpoint = "/api/constants"
	)

	expectedHeader := http.Header{
		headers.ContentType:     {"text/html; charset=iso-8859-1"},
		headers.ContentEncoding: {"deflate, gzip"},
	}

	middleware := cache.NewMiddleware(
		cache.WithCacheStorage(cache.NewRistrettoCache(1024*1024, time.Minute)),
		cache.WithMethods([]string{http.MethodGet}),
		cache.WithGlobs(config.CacheGlobs{
			"/translations",
			cacheGlob,
		}),
	)

	testHandler := testutils.NewCounter(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
		testutils.CopyHeaders(expectedHeader, writer.Header())
		fmt.Fprint(writer, expectedBody)
	}))

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
				testHandler.Reset()

				wrappedHandler := middleware.Wrap(testHandler)

				testutils.Times(5, func(_ int) {
					recorder := httptest.NewRecorder()
					rec := infra.NewResponseRecorder(recorder)
					wrappedHandler.ServeHTTP(
						rec,
						httptest.NewRequestWithContext(t.Context(), testCase.method, testCase.path, nil),
					)
					assert.Equal(t, expectedHeader, recorder.Header())
					assert.Equal(t, expectedBody, testutils.ReadBody(t, recorder))
				})

				assert.Equal(t, 1, testHandler.Count())
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
				path:       constantEndpoint,
				statusCode: http.StatusInternalServerError,
			},
			{
				name:       "witch response with status code 400",
				method:     http.MethodGet,
				path:       constantEndpoint,
				statusCode: http.StatusBadRequest,
			},
			{
				name:       "witch response with status code 304",
				method:     http.MethodGet,
				path:       constantEndpoint,
				statusCode: http.StatusNotModified,
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				testHandler := testutils.NewCounter(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
					writer.WriteHeader(testCase.statusCode)
					testutils.CopyHeaders(expectedHeader, writer.Header())
					fmt.Fprint(writer, expectedBody)
				}))

				wrappedHandler := middleware.Wrap(testHandler)

				testutils.Times(5, func(_ int) {
					recorder := httptest.NewRecorder()
					rec := infra.NewResponseRecorder(recorder)
					wrappedHandler.ServeHTTP(
						rec,
						httptest.NewRequestWithContext(t.Context(), testCase.method, testCase.path, nil),
					)
					assert.Equal(t, expectedHeader, recorder.Header())
					assert.Equal(t, expectedBody, testutils.ReadBody(t, recorder))
				})

				assert.Equal(t, 5, testHandler.Count())
			})
		}
	})

	t.Run("should not cache response between different hosts matched by one rule", func(t *testing.T) {
		const count = 5

		testHandler.Reset()

		middleware := cache.NewMiddleware(
			cache.WithCacheStorage(cache.NewRistrettoCache(1024*1024, time.Minute)),
			cache.WithMethods([]string{http.MethodGet}),
			cache.WithGlobs(config.CacheGlobs{cacheGlob}),
		)

		wrappedHandler := middleware.Wrap(testHandler)

		testutils.Times(count, func(index int) {
			recorder := httptest.NewRecorder()
			rec := infra.NewResponseRecorder(recorder)
			url := fmt.Sprintf("https://test-host-%d.com:4200/api/test", index)
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
			wrappedHandler.ServeHTTP(rec, request)
			assert.Equal(t, expectedHeader, recorder.Header())
			assert.Equal(t, expectedBody, testutils.ReadBody(t, recorder))
		})

		assert.Equal(t, count, testHandler.Count())
	})

	t.Run("should not cache response between different methods matched by one rule", func(t *testing.T) {
		methods := []string{http.MethodGet, http.MethodPost, http.MethodDelete, http.MethodPut}

		testHandler.Reset()

		middleware := cache.NewMiddleware(
			cache.WithCacheStorage(cache.NewRistrettoCache(1024*1024, time.Minute)),
			cache.WithMethods(methods),
			cache.WithGlobs(config.CacheGlobs{cacheGlob}),
		)

		testHandler := testutils.NewCounter(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusOK)
			testutils.CopyHeaders(expectedHeader, writer.Header())
			//nolint:gosec // G705: a test handler echoing the method it was called with
			fmt.Fprint(writer, methodBody(request.Method))
		}))

		wrappedHandler := middleware.Wrap(testHandler)

		for _, method := range methods {
			recorder := httptest.NewRecorder()
			rec := infra.NewResponseRecorder(recorder)
			request := httptest.NewRequestWithContext(t.Context(), method, "https://test-host.com:4200/api/test", nil)
			wrappedHandler.ServeHTTP(rec, request)
			assert.Equal(t, expectedHeader, recorder.Header())
			assert.Equal(t, methodBody(method), testutils.ReadBody(t, recorder))
		}

		assert.Len(t, methods, testHandler.Count())
	})
}
