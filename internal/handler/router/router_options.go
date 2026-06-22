package router

import (
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/di"
)

type Option = func(*Router)

func WithDiContainer(container *di.Container) Option {
	return func(r *Router) {
		r.container = container
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
