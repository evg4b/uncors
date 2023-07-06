package cache_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/cache"
	"github.com/evg4b/uncors/internal/sfmt"
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

func TestNewMiddleware(t *testing.T) {
	middleware := cache.NewMiddleware(
		cache.WithLogger(mocks.NewNoopLogger(t)),
		cache.WithGlobs([]string{"/demo"}),
	)

	const expectedBody = "this is test"

	t.Run("demo", func(t *testing.T) {
		handler := &test{Handler: func(writer contracts.ResponseWriter, request *contracts.Request) {
			writer.WriteHeader(http.StatusOK)
			sfmt.Fprintf(writer, expectedBody)
		}}

		wrappedHandler := middleware.Wrap(handler)

		callNTimes(5, func() {
			recorder := httptest.NewRecorder()
			wrappedRecorder := contracts.WrapResponseWriter(recorder)
			request := httptest.NewRequest(http.MethodGet, "/demo", nil)
			wrappedHandler.ServeHTTP(wrappedRecorder, request)
			assert.Equal(t, expectedBody, testutils.ReadBody(t, recorder))
		})

		assert.Equal(t, 1, handler.Count)
	})

	t.Run("demo", func(t *testing.T) {
		handler := &test{Handler: func(writer contracts.ResponseWriter, request *contracts.Request) {
			writer.WriteHeader(http.StatusOK)
			sfmt.Fprintf(writer, expectedBody)
		}}

		wrappedHandler := middleware.Wrap(handler)

		callNTimes(5, func() {
			recorder := httptest.NewRecorder()
			wrappedRecorder := contracts.WrapResponseWriter(recorder)
			request := httptest.NewRequest(http.MethodGet, "/test", nil)
			wrappedHandler.ServeHTTP(wrappedRecorder, request)
			assert.Equal(t, expectedBody, testutils.ReadBody(t, recorder))
		})

		assert.Equal(t, 5, handler.Count)
	})
}

type test struct {
	Handler func(writer contracts.ResponseWriter, request *contracts.Request)
	Count   int
}

func (t *test) ServeHTTP(writer contracts.ResponseWriter, request *contracts.Request) {
	t.Count++
	if t.Handler != nil {
		t.Handler(writer, request)
	}
}

func callNTimes(n int, function func()) {
	for i := 0; i < n; i++ {
		function()
	}
}
