package rewrite_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/rewrite"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/pkg/urlt"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMiddlewareWrap(t *testing.T) {
	t.Run("rewrites URL and calls next handler", func(t *testing.T) {
		expectedURL := "/rewritten"
		expectedHost := urlt.Host{Hostname: "example.com"}
		nextCalled := false

		middleware := rewrite.NewMiddleware(
			rewrite.WithRewritingOptions(&config.RewritingOption{
				To:   expectedURL,
				Host: expectedHost,
			}),
		)

		recorder := httptest.NewRecorder()
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/original", nil)
		helpers.NormaliseRequest(request)

		next := infra.HandlerFunc(func(_ contracts.ResponseWriter, request *contracts.Request) error {
			nextCalled = true

			assert.Equal(t, expectedURL, request.URL.Path)
			assert.Equal(t, expectedHost.HostPort(), request.Context().Value(rewrite.RewriteHostKey))

			return nil
		})

		handler := infra.Wrap(middleware, next)
		err := handler.ServeHTTP(server.NewResponseRecorder(recorder), request)
		require.NoError(t, err)

		assert.True(t, nextCalled)
	})

	t.Run("preserves the query string", func(t *testing.T) {
		middleware := rewrite.NewMiddleware(
			rewrite.WithRewritingOptions(&config.RewritingOption{
				From: "/api/v1/users",
				To:   "/v2/users",
			}),
		)

		recorder := httptest.NewRecorder()
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/users?page=2&sort=name", nil)
		helpers.NormaliseRequest(request)

		var rewritten string

		next := infra.HandlerFunc(func(_ contracts.ResponseWriter, request *contracts.Request) error {
			rewritten = request.URL.RequestURI()

			return nil
		})

		handler := infra.Wrap(middleware, next)
		require.NoError(t, handler.ServeHTTP(server.NewResponseRecorder(recorder), request))

		assert.Equal(t, "/v2/users?page=2&sort=name", rewritten)
	})

	t.Run("preserves original host when no host rewrite specified", func(t *testing.T) {
		expectedURL := "/rewritten"
		nextCalled := false

		middleware := rewrite.NewMiddleware(
			rewrite.WithRewritingOptions(&config.RewritingOption{
				To: expectedURL,
			}),
		)

		recorder := httptest.NewRecorder()
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/original", nil)
		helpers.NormaliseRequest(request)

		next := infra.HandlerFunc(func(_ contracts.ResponseWriter, request *contracts.Request) error {
			nextCalled = true

			assert.Equal(t, expectedURL, request.URL.Path)
			assert.Nil(t, request.Context().Value(rewrite.RewriteHostKey))

			return nil
		})

		handler := infra.Wrap(middleware, next)
		serveErr := handler.ServeHTTP(server.NewResponseRecorder(recorder), request)
		require.NoError(t, serveErr)

		assert.True(t, nextCalled)
	})
}

func TestMiddlewareSubstitutesPathVariablesOnce(t *testing.T) {
	tests := []struct {
		name string
		to   string
		vars map[string]string
		want string
	}{
		{
			name: "value that looks like another placeholder",
			to:   "/api/{name}/{id}",
			vars: map[string]string{"id": "{name}", "name": "bob"},
			want: "/api/bob/{name}",
		},
		{
			name: "unknown placeholder stays",
			to:   "/api/{missing}/{id}",
			vars: map[string]string{"id": "7"},
			want: "/api/{missing}/7",
		},
		{
			name: "nested braces",
			to:   "/api/{{id}}",
			vars: map[string]string{"id": "7"},
			want: "/api/{7}",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			middleware := rewrite.NewMiddleware(
				rewrite.WithRewritingOptions(&config.RewritingOption{To: testCase.to}),
			)

			for range 50 {
				request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/original", nil)
				helpers.NormaliseRequest(request)
				request = mux.SetURLVars(request, testCase.vars)

				var path string

				next := infra.HandlerFunc(func(_ contracts.ResponseWriter, request *contracts.Request) error {
					path = request.URL.Path

					return nil
				})

				err := infra.Wrap(middleware, next).ServeHTTP(server.NewResponseRecorder(httptest.NewRecorder()), request)
				require.NoError(t, err)
				assert.Equal(t, testCase.want, path)
			}
		})
	}
}
