package rewrite_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/rewrite"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/pkg/urlt"
	"github.com/stretchr/testify/assert"
)

func TestMiddlewareWrap(t *testing.T) {
	t.Run("rewrites URL and calls next handler", func(t *testing.T) {
		expectedURL := "/rewritten"
		expectedHost := urlt.Host{Hostname: "example.com"}
		nextCalled := false

		middleware := rewrite.NewMiddleware(
			rewrite.WithRewritingOptions(config.RewritingOption{
				To:   expectedURL,
				Host: expectedHost,
			}),
		)

		recorder := httptest.NewRecorder()
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/original", nil)
		helpers.NormaliseRequest(request)

		next := contracts.HandlerFunc(func(_ contracts.ResponseWriter, request *contracts.Request) error {
			nextCalled = true

			assert.Equal(t, expectedURL, request.URL.Path)
			assert.Equal(t, expectedHost.HostPort(), request.Context().Value(rewrite.RewriteHostKey))

			return nil
		})

		handler := middleware.Wrap(next)
		handler.ServeHTTP(contracts.NewResponseRecorder(recorder), request) //nolint:errcheck

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
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/original", nil)
		helpers.NormaliseRequest(request)

		next := contracts.HandlerFunc(func(_ contracts.ResponseWriter, request *contracts.Request) error {
			nextCalled = true

			assert.Equal(t, expectedURL, request.URL.Path)
			assert.Nil(t, request.Context().Value(rewrite.RewriteHostKey))

			return nil
		})

		handler := middleware.Wrap(next)
		handler.ServeHTTP(contracts.NewResponseRecorder(recorder), request) //nolint:errcheck

		assert.True(t, nextCalled)
	})
}
