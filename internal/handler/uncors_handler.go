package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/evg4b/uncors/internal/infra"

	"github.com/evg4b/uncors/pkg/urlx"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/mock"
	"github.com/evg4b/uncors/internal/handler/proxy"
	"github.com/evg4b/uncors/internal/ui"
	"github.com/gorilla/mux"
	"github.com/spf13/afero"
)

type RequestHandler struct {
	router          *mux.Router
	fs              afero.Fs
	logger          contracts.Logger
	mappings        []config.URLMapping
	replacerFactory contracts.URLReplacerFactory
	httpClient      contracts.HTTPClient
}

var errHostNotMapped = errors.New("host not mapped")

func NewUncorsRequestHandler(options ...UncorsRequestHandlerOption) *RequestHandler {
	handler := &RequestHandler{
		router:   mux.NewRouter(),
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

	for _, mapping := range handler.mappings {
		uri, err := urlx.Parse(mapping.From)
		if err != nil {
			panic(err)
		}

		host, _, err := urlx.SplitHostPort(uri)
		if err != nil {
			panic(err)
		}

		router := handler.router.Host(replaceWildcards(host)).Subrouter()

		handler.makeStaticRoutes(router, mapping.Statics, proxyHandler)
		handler.makeMockedRoutes(router, mapping.Mocks)
		setDefaultHandler(router, proxyHandler)
	}

	setDefaultHandler(handler.router, http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		infra.HTTPError(writer, errHostNotMapped)
	}))

	return handler
}

func (m *RequestHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	m.router.ServeHTTP(writer, request)
}

func (m *RequestHandler) createHandler(response config.Response) *mock.Middleware {
	return mock.NewMockMiddleware(
		mock.WithLogger(ui.MockLogger),
		mock.WithResponse(response),
		mock.WithFileSystem(m.fs),
	)
}

func setDefaultHandler(router *mux.Router, handler http.Handler) {
	router.NotFoundHandler = handler
	router.MethodNotAllowedHandler = handler
}

const wildcard = "*"

func replaceWildcards(host string) string {
	count := strings.Count(host, wildcard)
	for i := 1; i <= count; i++ {
		host = strings.Replace(host, wildcard, fmt.Sprintf("{p%d}", i), 1)
	}

	return host
}
