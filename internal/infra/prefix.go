package infra

import (
	"context"

	"github.com/evg4b/uncors/internal/contracts"
)

func WithPrefix(prefix string, next contracts.Handler) contracts.Handler {
	return HandlerFunc(func(resp contracts.ResponseWriter, req *contracts.Request) error {
		if updater, ok := req.Context().Value(contracts.PrefixUpdaterKey).(func(string)); ok {
			updater(prefix)
		}

		ctx := context.WithValue(req.Context(), contracts.PrefixKey, prefix)

		return next.ServeHTTP(resp, req.WithContext(ctx))
	})
}

type PrefixedMiddleware struct {
	middleware contracts.Middleware
	prefix     string
}

func NewPrefixedMiddleware(middleware contracts.Middleware, prefix string) *PrefixedMiddleware {
	return &PrefixedMiddleware{
		middleware: middleware,
		prefix:     prefix,
	}
}

func (p *PrefixedMiddleware) ServeHTTP(w contracts.ResponseWriter, r *contracts.Request, next contracts.Next) error {
	return p.middleware.ServeHTTP(w, r, func(w contracts.ResponseWriter, r *contracts.Request) error {
		return WithPrefix(p.prefix, HandlerFunc(next)).
			ServeHTTP(w, r)
	})
}
