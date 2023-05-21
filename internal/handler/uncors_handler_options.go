package handler

import (
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/spf13/afero"
)

type UncorsRequestHandlerOption = func(*RequestHandler)

func WithLogger(logger contracts.Logger) UncorsRequestHandlerOption {
	return func(m *RequestHandler) {
		m.logger = logger
	}
}

func WithFileSystem(fs afero.Fs) UncorsRequestHandlerOption {
	return func(m *RequestHandler) {
		m.fs = fs
	}
}

func WithURLReplacerFactory(replacerFactory contracts.URLReplacerFactory) UncorsRequestHandlerOption {
	return func(m *RequestHandler) {
		m.replacerFactory = replacerFactory
	}
}

func WithHTTPClient(client contracts.HTTPClient) UncorsRequestHandlerOption {
	return func(m *RequestHandler) {
		m.httpClient = client
	}
}

func WithMappings(mappings []config.URLMapping) UncorsRequestHandlerOption {
	return func(m *RequestHandler) {
		m.mappings = mappings
	}
}
