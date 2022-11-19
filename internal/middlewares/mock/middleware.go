package mock

import (
	"net/http"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/gorilla/mux"
)

type Middleware struct {
	router *mux.Router
	next   http.Handler
	logger contracts.Logger
	mocks  []Mock
}

func NewMockMiddleware(options ...MiddlewareOption) *Middleware {
	router := mux.NewRouter()
	middleware := &Middleware{router: router, mocks: []Mock{}}

	for _, option := range options {
		option(middleware)
	}

	makeMockedRoutes(middleware.router, middleware.logger, middleware.mocks)
	router.NotFoundHandler = middleware.next
	router.MethodNotAllowedHandler = middleware.next

	return middleware
}

func (m *Middleware) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	m.router.ServeHTTP(response, request)
}
