package proxy

import (
	"net/http"

	"github.com/evg4b/uncors/internal/contracts"
)

type MiddlewareOption = func(*Middleware)

func WithURLReplacerFactory(replacerFactory contracts.URLReplacerFactory) MiddlewareOption {
	return func(m *Middleware) {
		m.replacers = replacerFactory
	}
}

func WithHTTPClient(http *http.Client) MiddlewareOption {
	return func(m *Middleware) {
		m.http = http
	}
}

func WithLogger(logger contracts.Logger) MiddlewareOption {
	return func(m *Middleware) {
		m.logger = logger
	}
}
