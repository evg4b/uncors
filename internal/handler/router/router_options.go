package router

import (
	"github.com/evg4b/uncors/internal/contracts"
)

type Option = func(*Router)

func WithFactory(factory Factory) Option {
	return func(r *Router) {
		r.factory = factory
	}
}

func ForRouterWithCacheMiddlewareFactory(factory CacheMiddlewareFactory) Option {
	return func(r *Router) {
		r.cacheMiddlewareFactory = factory
	}
}

func ForRouterWithDefaultHandler(handler contracts.Handler) Option {
	return func(r *Router) {
		r.defaultHandler = handler
	}
}
