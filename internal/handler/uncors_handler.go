package handler

import (
	"net/http"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/mock"
	"github.com/evg4b/uncors/internal/handler/proxy"
	"github.com/evg4b/uncors/internal/ui"
	"github.com/gorilla/mux"
	"github.com/spf13/afero"
)

type UncorsRequestHandler struct {
	router          *mux.Router
	fs              afero.Fs
	logger          contracts.Logger
	mocks           []config.Mock
	mappings        []config.URLMapping
	replacerFactory contracts.URLReplacerFactory
	httpClient      contracts.HTTPClient
}

func NewUncorsRequestHandler(options ...UncorsRequestHandlerOption) *UncorsRequestHandler {
	handler := &UncorsRequestHandler{
		router:   mux.NewRouter(),
		mocks:    []config.Mock{},
		mappings: []config.URLMapping{},
	}

	for _, option := range options {
		option(handler)
	}

	proxyHandler := proxy.NewProxyHandler(
		proxy.WithURLReplacerFactory(handler.replacerFactory),
		proxy.WithHTTPClient(handler.httpClient),
		proxy.WithLogger(ui.ProxyLogger),
	)

	handler.makeMockedRoutes()
	handler.makeStaticRoutes(proxyHandler)
	handler.setDefaultHandler(proxyHandler)

	return handler
}

func (m *UncorsRequestHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	m.router.ServeHTTP(writer, request)
}

func (m *UncorsRequestHandler) createHandler(response config.Response) *mock.Middleware {
	return mock.NewMockMiddleware(
		mock.WithLogger(ui.MockLogger),
		mock.WithResponse(response),
		mock.WithFileSystem(m.fs),
	)
}

func (m *UncorsRequestHandler) setDefaultHandler(handler http.Handler) {
	m.router.NotFoundHandler = handler
	m.router.MethodNotAllowedHandler = handler
}
