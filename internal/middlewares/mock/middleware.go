package mock

import (
	"net/http"

	"github.com/spf13/afero"

	"github.com/evg4b/uncors/internal/configuration"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/gorilla/mux"
)

type Middleware struct {
	router   *mux.Router
	fs       afero.Fs
	logger   contracts.Logger
	next     http.Handler
	mocks    []configuration.Mock
	mappings []configuration.URLMapping
}

func NewMockMiddleware(options ...MiddlewareOption) *Middleware {
	router := mux.NewRouter()
	middleware := &Middleware{
		router:   router,
		mocks:    []configuration.Mock{},
		mappings: []configuration.URLMapping{},
	}

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
