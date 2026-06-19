package server

import (
	"net/http"

	"github.com/evg4b/uncors/internal/contracts"
)

// ResponseRecorder is an alias for contracts.ResponseRecorder kept here so
// middleware tests can import from server without a circular dependency.
type ResponseRecorder = contracts.ResponseRecorder

func NewResponseRecorder(w http.ResponseWriter) *ResponseRecorder {
	return contracts.NewResponseRecorder(w)
}
