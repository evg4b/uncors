package proxy

import (
	"net/http"

	"github.com/evg4b/uncors/internal/contracts"
)

type HandlerOption = func(*Handler)

func WithURLReplacerFactory(replacerFactory contracts.URLReplacerFactory) HandlerOption {
	return func(pm *Handler) {
		pm.replacers = replacerFactory
	}
}

func WithHTTPClient(http *http.Client) HandlerOption {
	return func(pm *Handler) {
		pm.http = http
	}
}

func WithLogger(logger contracts.Logger) HandlerOption {
	return func(pm *Handler) {
		pm.logger = logger
	}
}
