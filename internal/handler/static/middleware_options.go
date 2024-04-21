package static

import (
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/tui/monitor"
	"github.com/spf13/afero"
)

type MiddlewareOption = func(*Middleware)

func WithFileSystem(fs afero.Fs) MiddlewareOption {
	return func(h *Middleware) {
		h.fs = fs
	}
}

func WithIndex(index string) MiddlewareOption {
	return func(h *Middleware) {
		h.index = index
	}
}

func WithLogger(logger contracts.Logger) MiddlewareOption {
	return func(h *Middleware) {
		h.logger = logger
	}
}

func WithPrefix(prefix string) MiddlewareOption {
	return func(h *Middleware) {
		h.prefix = prefix
	}
}

func WithRequestTracker(tracker monitor.RequestTracker) MiddlewareOption {
	return func(h *Middleware) {
		h.tracker = tracker
	}
}
