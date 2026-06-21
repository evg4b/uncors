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

func ForRouterWithStaticMiddlewareFactory(factory StaticMiddlewareFactory) RouterOption {
	return func(r *Router) {
		r.staticMiddlewareFactory = factory
	}
}

func ForRouterWithMockHandlerFactory(factory MockHandlerFactory) RouterOption {
	return func(r *Router) {
		r.mockHandlerFactory = factory
	}
}

func ForRouterWithScriptHandlerFactory(factory ScriptHandlerFactory) RouterOption {
	return func(r *Router) {
		r.scriptHandlerFactory = factory
	}
}

func ForRouterWithRewriteMiddlewareFactory(factory RewriteMiddlewareFactory) RouterOption {
	return func(r *Router) {
		r.rewriteMiddlewareFactory = factory
	}
}

func ForRouterWithOptionsMiddlewareFactory(factory OptionsMiddlewareFactory) RouterOption {
	return func(r *Router) {
		r.optionsMiddlewareFactory = factory
	}
}

func ForRouterWithHARMiddlewareFactory(factory HARMiddlewareFactory) RouterOption {
	return func(r *Router) {
		r.harMiddlewareFactory = factory
	}
}

func ForRouterWithDefaultHandler(handler contracts.Handler) RouterOption {
	return func(r *Router) {
		r.defaultHandler = handler
	}
}
