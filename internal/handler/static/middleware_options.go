package static

import (
	"net/http"

	"github.com/spf13/afero"
)

type MiddlewareOption = func(*Middleware)

func WithFileSystem(fs afero.Fs) MiddlewareOption {
	return func(m *Middleware) {
		m.fs = fs
	}
}

func WithDir(prefix string) MiddlewareOption {
	return func(m *Middleware) {
		m.prefix = prefix
	}
}

func WithNext(next http.Handler) MiddlewareOption {
	return func(m *Middleware) {
		m.next = next
	}
}
