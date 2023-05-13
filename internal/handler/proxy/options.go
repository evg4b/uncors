package proxy

import (
	"github.com/evg4b/uncors/internal/contracts"
)

type HandlerOption = func(*Handler)

func WithURLReplacerFactory(replacerFactory contracts.URLReplacerFactory) HandlerOption {
	return func(m *Handler) {
		m.replacers = replacerFactory
	}
}

func WithHTTPClient(http contracts.HTTPClient) HandlerOption {
	return func(m *Handler) {
		m.http = http
	}
}

func WithLogger(logger contracts.Logger) HandlerOption {
	return func(m *Handler) {
		m.logger = logger
	}
}
