package cache_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/cache"
	"github.com/evg4b/uncors/internal/sfmt"
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/go-http-utils/headers"
	goCache "github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
)

func TestNewMiddleware(t *testing.T) {
	const expectedBody = "this is test"
	expectedHeader := http.Header{
		headers.ContentType:     {"text/html; charset=iso-8859-1"},
		headers.ContentEncoding: {"deflate, gzip"},
	}

	middleware := cache.NewMiddleware(
		cache.WithCacheStorage(goCache.New(time.Minute, time.Minute)),
		cache.WithLogger(mocks.NewNoopLogger(t)),
		cache.WithMethods(http.MethodGet),
		cache.WithGlobs(
			"/translations",
			"/api/**",
		),
	)

	t.Run("should not call cached response just one time for", func(t *testing.T) {
		tests := []struct {
			name   string
			method string
			path   string
		}{
			{name: "request in glob", method: http.MethodGet, path: "/api"},
			{name: "request in glob with params", method: http.MethodGet, path: "/api?some=params"},
			{name: "request in glob with other params", method: http.MethodGet, path: "/api?other=params"},
			{name: "second level request in glob", method: http.MethodGet, path: "/api/comments"},
			{name: "second level request in glob with params", method: http.MethodGet, path: "/api/comments?q=test"},
			{name: "third level request in glob", method: http.MethodGet, path: "/api/comments/1"},
			{name: "third level request in glob with params", method: http.MethodGet, path: "/api/comments/1?q=demo"},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				handler := testutils.NewCounter(func(writer contracts.ResponseWriter, request *contracts.Request) {
					writer.WriteHeader(http.StatusOK)
					testutils.CopyHeaders(expectedHeader, writer.Header())
					sfmt.Fprintf(writer, expectedBody)
				})

				wrappedHandler := middleware.Wrap(handler)

				testutils.Times(5, func() {
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
			name   string
			method string
			path   string
		}{
			{name: "with path out of glob", method: http.MethodGet, path: "/test"},
			{name: "with POST method", method: http.MethodPost, path: "/api"},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				handler := testutils.NewCounter(func(writer contracts.ResponseWriter, request *contracts.Request) {
					writer.WriteHeader(http.StatusOK)
					testutils.CopyHeaders(expectedHeader, writer.Header())
					sfmt.Fprintf(writer, expectedBody)
				})

				wrappedHandler := middleware.Wrap(handler)

				testutils.Times(5, func() {
					recorder := httptest.NewRecorder()
					wrappedHandler.ServeHTTP(
						contracts.WrapResponseWriter(recorder),
						httptest.NewRequest(http.MethodGet, "/test", nil),
					)
					assert.Equal(t, expectedHeader, recorder.Header())
					assert.Equal(t, expectedBody, testutils.ReadBody(t, recorder))
				})

				assert.Equal(t, 5, handler.Count())
			})
		}
	})
}
