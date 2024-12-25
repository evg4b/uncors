package proxy

import (
	"github.com/evg4b/uncors/internal/config"
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

func WithLogger(logger contracts.Logger) HandlerOption {
	return func(m *Handler) {
		m.logger = logger
	}
}

type RewriteOption = func(*RwreiteHandler)

func WithRewritingOptions(rewrite config.RewritingOption) RewriteOption {
	return func(h *RwreiteHandler) {
		h.rewrite = rewrite
	}
}

func WithHTTPClientR(http contracts.HTTPClient) RewriteOption {
	return func(h *RwreiteHandler) {
		h.http = http
	}
}

func WithLoggerR(logger contracts.Logger) RewriteOption {
	return func(h *RwreiteHandler) {
		h.logger = logger
	}
}
