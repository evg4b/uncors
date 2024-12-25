package proxy

import (
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/urlreplacer"
)

type HandlerOption = func(*Handler)

func WithURLReplacerFactory(replacerFactory urlreplacer.ReplacerFactory) HandlerOption {
	return func(m *Handler) {
		m.replacers = replacerFactory
	}
}

func WithHTTPClient(http contracts.HTTPClient) HandlerOption {
	return func(m *Handler) {
		m.http = http
	}
}

func WithProxyLogger(logger contracts.Logger) HandlerOption {
	return func(m *Handler) {
		m.proxyLogger = logger
	}
}

func WithRewriteLogger(logger contracts.Logger) HandlerOption {
	return func(m *Handler) {
		m.rewriteLogger = logger
	}
}
