package handler

import (
	"github.com/evg4b/uncors/internal/configuration"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/spf13/afero"
)

type UncorsRequestHandlerOption = func(*UncorsRequestHandler)

func WithLogger(logger contracts.Logger) UncorsRequestHandlerOption {
	return func(m *UncorsRequestHandler) {
		m.logger = logger
	}
}

func WithMocks(mocks []configuration.Mock) UncorsRequestHandlerOption {
	return func(m *UncorsRequestHandler) {
		m.mocks = mocks
	}
}

func WithFileSystem(fs afero.Fs) UncorsRequestHandlerOption {
	return func(m *UncorsRequestHandler) {
		m.fs = fs
	}
}

func WithURLReplacerFactory(replacerFactory contracts.URLReplacerFactory) UncorsRequestHandlerOption {
	return func(m *UncorsRequestHandler) {
		m.replacerFactory = replacerFactory
	}
}

func WithHTTPClient(client contracts.HTTPClient) UncorsRequestHandlerOption {
	return func(m *UncorsRequestHandler) {
		m.httpClient = client
	}
}

func WithMappings(mappings []configuration.URLMapping) UncorsRequestHandlerOption {
	return func(m *UncorsRequestHandler) {
		m.mappings = mappings
	}
}
