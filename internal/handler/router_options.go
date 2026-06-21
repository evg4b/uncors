package handler

import (
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/di"
)

type RouterOption = func(*Router)

func WithDiContainer(container *di.Container) RouterOption {
	return func(r *Router) {
		r.container = container
	}
}

func ForRouterWithCacheMiddlewareFactory(factory CacheMiddlewareFactory) RouterOption {
	return func(r *Router) {
		r.cacheMiddlewareFactory = factory
	}
}

func ForRouterWithDefaultHandler(handler contracts.Handler) RouterOption {
	return func(r *Router) {
		r.defaultHandler = handler
	}
}
