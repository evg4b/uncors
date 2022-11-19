package mock

import (
	"net/http"

	"github.com/evg4b/uncors/internal/contracts"
)

type MiddlewareOption = func(*Middleware)

func WithLogger(logger contracts.Logger) MiddlewareOption {
	return func(m *Middleware) {
		m.logger = logger
	}
}

func WithNextMiddleware(next http.Handler) MiddlewareOption {
	return func(m *Middleware) {
		m.next = next
	}
}

func WithMocks(mocks []Mock) MiddlewareOption {
	return func(m *Middleware) {
		m.mocks = mocks
	}
}
