package infra

import (
	"context"
	"net/http"

	"github.com/evg4b/uncors/internal/contracts"
)

// WithPrefix labels every request served by next with prefix, so that the
// activity log can show which handler answered it.
func WithPrefix(prefix string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if updater, ok := request.Context().Value(contracts.PrefixUpdaterKey).(func(string)); ok {
			updater(prefix)
		}

		ctx := context.WithValue(request.Context(), contracts.PrefixKey, prefix)

		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

// NewPrefixedMiddleware labels the requests that middleware passes through to
// the rest of the chain. Requests the middleware answers itself keep the prefix
// of whoever answered them.
func NewPrefixedMiddleware(middleware contracts.Middleware, prefix string) contracts.Middleware {
	return func(next http.Handler) http.Handler {
		return middleware(WithPrefix(prefix, next))
	}
}
