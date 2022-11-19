package proxy

import (
	"net/http"

	"github.com/evg4b/uncors/internal/contracts"
)

type MiddelwareOption = func(*Middelware)

func WithURLReplacerFactory(replacerFactory contracts.URLReplacerFactory) MiddelwareOption {
	return func(m *Middelware) {
		m.replacers = replacerFactory
	}
}

func WithHTTPClient(http *http.Client) MiddelwareOption {
	return func(m *Middelware) {
		m.http = http
	}
}

func WithLogger(logger contracts.Logger) MiddelwareOption {
	return func(m *Middelware) {
		m.logger = logger
	}
}
