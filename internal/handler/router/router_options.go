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

func WithCacheMiddlewareFactory(factory CacheMiddlewareFactory) Option {
	return func(r *Router) {
		r.cacheMiddlewareFactory = factory
	}
}

func WithDefaultHandler(handler contracts.Handler) Option {
	return func(r *Router) {
		r.defaultHandler = handler
	}
}
