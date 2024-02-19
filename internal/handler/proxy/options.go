package proxy

import (
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/tui"
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

func WithRequestTracker(tracker tui.RequestTracker) HandlerOption {
	return func(m *Handler) {
		m.tracker = tracker
	}
}
