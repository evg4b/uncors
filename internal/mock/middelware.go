package mock

import (
	"net/http"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/gorilla/mux"
)

type Middelware struct {
	router *mux.Router
	next   http.Handler
	logger contracts.Logger
	mocks  []Mock
}

func NewMockMiddelware(options ...MiddelwareOption) *Middelware {
	router := mux.NewRouter()
	middelware := &Middelware{router: router, mocks: []Mock{}}

	for _, option := range options {
		option(middelware)
	}

	makeMockedRoutes(middelware.router, middelware.logger, middelware.mocks)
	router.NotFoundHandler = middelware.next
	router.MethodNotAllowedHandler = middelware.next

	return middelware
}

func (m *Middelware) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	m.router.ServeHTTP(response, request)
}
