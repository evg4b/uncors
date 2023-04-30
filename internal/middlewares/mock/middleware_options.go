package mock

import (
	"net/http"

	"github.com/evg4b/uncors/internal/configuration"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/spf13/afero"
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

func WithMocks(mocks []configuration.Mock) MiddlewareOption {
	return func(m *Middleware) {
		m.mocks = mocks
	}
}

func WithFileSystem(fs afero.Fs) MiddlewareOption {
	return func(m *Middleware) {
		m.fs = fs
	}
}
