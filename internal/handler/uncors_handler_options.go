package handler

import (
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/urlreplacer"
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

func WithURLReplacerFactory(replacerFactory urlreplacer.ReplacerFactory) RequestHandlerOption {
	return func(h *RequestHandler) {
		h.replacerFactory = replacerFactory
	}
}

func WithHTTPClient(client contracts.HTTPClient) RequestHandlerOption {
	return func(h *RequestHandler) {
		h.httpClient = client
	}
}

func WithMappings(mappings config.Mappings) RequestHandlerOption {
	return func(h *RequestHandler) {
		h.mappings = mappings
	}
}

type cacheMiddlewareFactory = func(key string, globs config.CacheGlobs) contracts.Middleware

func WithCacheMiddlewareFactory(factory cacheMiddlewareFactory) RequestHandlerOption {
	return func(h *RequestHandler) {
		h.cacheMiddlewareFactory = factory
	}
}
