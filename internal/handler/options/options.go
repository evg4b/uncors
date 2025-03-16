package options

import (
	"github.com/evg4b/uncors/internal/contracts"
)

type MiddlewareOption = func(*Middleware)

func WithLogger(logger contracts.Logger) MiddlewareOption {
	return func(m *Middleware) {
		m.logger = logger
	}
}

func WithHeaders(headers map[string]string) MiddlewareOption {
	return func(m *Middleware) {
		m.headers = headers
	}
}

func WithCode(code uint) MiddlewareOption {
	return func(m *Middleware) {
		m.code = code
	}
}
