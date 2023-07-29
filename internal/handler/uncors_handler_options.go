package handler

import (
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/spf13/afero"
)

type RequestHandlerOption = func(*RequestHandler)

func WithLogger(logger contracts.Logger) RequestHandlerOption {
	return func(h *RequestHandler) {
		h.logger = logger
	}
}

func WithFileSystem(fs afero.Fs) RequestHandlerOption {
	return func(h *RequestHandler) {
		h.fs = fs
	}
}

func WithMappings(mappings config.Mappings) RequestHandlerOption {
	return func(h *RequestHandler) {
		h.mappings = mappings
	}
}

func WithCacheMiddlewareFactory(factory cacheMiddlewareFactory) RequestHandlerOption {
	return func(h *RequestHandler) {
		h.cacheMiddlewareFactory = factory
	}
}

func WithProxyHandlerFactory(factory proxyHandlerFactory) RequestHandlerOption {
	return func(h *RequestHandler) {
		h.proxyHandlerFactory = factory
	}
}
