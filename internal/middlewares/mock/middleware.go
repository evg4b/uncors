package mock

import (
	"net/http"

	"github.com/evg4b/uncors/internal/configuration"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/gorilla/mux"
	"github.com/spf13/afero"
)

type Middleware struct {
	router   *mux.Router
	next     http.Handler
	logger   contracts.Logger
	mocks    []configuration.Mock
	fs       afero.Fs
	mappings []configuration.URLMapping
}

func NewMockMiddleware(options ...MiddlewareOption) *Middleware {
	router := mux.NewRouter()
	middleware := &Middleware{router: router, mocks: []configuration.Mock{}}

	for _, option := range options {
		option(middleware)
	}

	middleware.makeStaticRoutes()
	middleware.makeMockedRoutes()
	router.NotFoundHandler = middleware.next
	router.MethodNotAllowedHandler = middleware.next

	return middleware
}

func (m *Middleware) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	m.router.ServeHTTP(response, request)
}
