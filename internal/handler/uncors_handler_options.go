package handler

import (
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/spf13/afero"
)

type UncorsRequestHandlerOption = func(*RequestHandler)

func WithLogger(logger contracts.Logger) UncorsRequestHandlerOption {
	return func(h *RequestHandler) {
		h.logger = logger
	}
}

func WithFileSystem(fs afero.Fs) UncorsRequestHandlerOption {
	return func(h *RequestHandler) {
		h.fs = fs
	}
}

func WithURLReplacerFactory(replacerFactory contracts.URLReplacerFactory) UncorsRequestHandlerOption {
	return func(h *RequestHandler) {
		h.replacerFactory = replacerFactory
	}
}

func WithHTTPClient(client contracts.HTTPClient) UncorsRequestHandlerOption {
	return func(h *RequestHandler) {
		h.httpClient = client
	}
}

func WithMappings(mappings config.Mappings) UncorsRequestHandlerOption {
	return func(h *RequestHandler) {
		h.mappings = mappings
	}
}
