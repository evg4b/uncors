package static

import (
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/spf13/afero"
)

type MiddlewareOption = func(*Middleware)

func WithFileSystem(fs afero.Fs) MiddlewareOption {
	return func(m *Middleware) {
		m.fs = fs
	}
}

func WithIndex(index string) MiddlewareOption {
	return func(m *Middleware) {
		m.index = index
	}
}

func WithNext(next contracts.Handler) MiddlewareOption {
	return func(m *Middleware) {
		m.next = next
	}
}

func WithLogger(logger contracts.Logger) MiddlewareOption {
	return func(m *Middleware) {
		m.logger = logger
	}
}

func WithPrefix(prefix string) MiddlewareOption {
	return func(m *Middleware) {
		m.prefix = prefix
	}
}
