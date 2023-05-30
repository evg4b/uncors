package mock

import (
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/spf13/afero"
)

type MiddlewareOption = func(*Middleware)

func WithLogger(logger contracts.Logger) MiddlewareOption {
	return func(m *Middleware) {
		m.logger = logger
	}
}

func WithResponse(response config.Response) MiddlewareOption {
	return func(m *Middleware) {
		m.response = response
	}
}

func WithFileSystem(fs afero.Fs) MiddlewareOption {
	return func(m *Middleware) {
		m.fs = fs
	}
}

func WithAfter(after func(duration time.Duration) <-chan time.Time) MiddlewareOption {
	return func(m *Middleware) {
		m.after = after
	}
}
