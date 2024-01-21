package handler

import (
	"github.com/charmbracelet/log"
	"github.com/evg4b/uncors/internal/config"
)

type RequestHandlerOption = func(*RequestHandler)

func WithLogger(logger *log.Logger) RequestHandlerOption {
	return func(h *RequestHandler) {
		h.logger = logger
	}
}

func WithMappings(mappings config.Mappings) RequestHandlerOption {
	return func(h *RequestHandler) {
		h.mappings = mappings
	}
}

func WithCacheMiddlewareFactory(factory CacheMiddlewareFactory) RequestHandlerOption {
	return func(h *RequestHandler) {
		h.cacheMiddlewareFactory = factory
	}
}

func WithProxyHandlerFactory(factory ProxyHandlerFactory) RequestHandlerOption {
	return func(h *RequestHandler) {
		h.proxyHandlerFactory = factory
	}
}

func WithStaticHandlerFactory(factory StaticMiddlewareFactory) RequestHandlerOption {
	return func(h *RequestHandler) {
		h.staticMiddlewareFactory = factory
	}
}

func WithMockHandlerFactory(factory MockHandlerFactory) RequestHandlerOption {
	return func(h *RequestHandler) {
		h.mockHandlerFactory = factory
	}
}
