package rewrite_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/rewrite"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/stretchr/testify/assert"
)

func TestMiddlewareWrap(t *testing.T) {
	t.Run("rewrites URL and calls next handler", func(t *testing.T) {
		expectedURL := "/rewritten"
		expectedHost := "example.com"
		nextCalled := false

		middleware := rewrite.NewMiddleware(
			rewrite.WithRewritingOptions(config.RewritingOption{
				To:   expectedURL,
				Host: expectedHost,
			}),
		)

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/original", nil)
		helpers.NormaliseRequest(request)

		next := contracts.HandlerFunc(func(_ contracts.ResponseWriter, request *contracts.Request) {
			nextCalled = true
			assert.Equal(t, expectedURL, request.URL.Path)
			assert.Equal(t, expectedHost, request.Context().Value(rewrite.RewriteHostKey))
		})

		handler := middleware.Wrap(next)
		handler.ServeHTTP(contracts.WrapResponseWriter(recorder), request)

		assert.True(t, nextCalled)
	})

	t.Run("preserves original host when no host rewrite specified", func(t *testing.T) {
		expectedURL := "/rewritten"
		nextCalled := false

		middleware := rewrite.NewMiddleware(
			rewrite.WithRewritingOptions(config.RewritingOption{
				To: expectedURL,
			}),
		)

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/original", nil)
		helpers.NormaliseRequest(request)

		next := contracts.HandlerFunc(func(_ contracts.ResponseWriter, request *contracts.Request) {
			nextCalled = true
			assert.Equal(t, expectedURL, request.URL.Path)
			assert.Nil(t, request.Context().Value(rewrite.RewriteHostKey))
		})

		handler := middleware.Wrap(next)
		handler.ServeHTTP(contracts.WrapResponseWriter(recorder), request)

		assert.True(t, nextCalled)
	})
}
