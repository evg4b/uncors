package mock

import (
	"net/http"

	"github.com/evg4b/uncors/internal/contracts"
)

type MiddelwareOption = func(*Middelware)

func WithLogger(logger contracts.Logger) MiddelwareOption {
	return func(m *Middelware) {
		m.logger = logger
	}
}

func WithNextMiddelware(next http.Handler) MiddelwareOption {
	return func(m *Middelware) {
		m.next = next
	}
}

func WithMocks(mocks []Mock) MiddelwareOption {
	return func(m *Middelware) {
		m.mocks = mocks
	}
}
