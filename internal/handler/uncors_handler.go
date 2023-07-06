package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/mock"
	"github.com/evg4b/uncors/internal/handler/proxy"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/sfmt"
	"github.com/evg4b/uncors/internal/ui"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/evg4b/uncors/pkg/urlx"
	"github.com/gorilla/mux"
	"github.com/spf13/afero"
)

type RequestHandler struct {
	router                 *mux.Router
	fs                     afero.Fs
	logger                 contracts.Logger
	mappings               config.Mappings
	replacerFactory        urlreplacer.ReplacerFactory
	httpClient             contracts.HTTPClient
	cacheMiddlewareFactory cacheMiddlewareFactory
}

var errHostNotMapped = errors.New("host not mapped")

func NewUncorsRequestHandler(options ...RequestHandlerOption) *RequestHandler {
	handler := &RequestHandler{
		router:   mux.NewRouter(),
		mappings: config.Mappings{},
	}

	for _, option := range options {
		option(handler)
	}

	helpers.AssertIsDefined(handler.cacheMiddlewareFactory, "Cache middleware is not set")

	proxyHandler := proxy.NewProxyHandler(
		proxy.WithURLReplacerFactory(handler.replacerFactory),
		proxy.WithHTTPClient(handler.httpClient),
		proxy.WithLogger(ui.ProxyLogger),
	)

	for index, mapping := range handler.mappings {
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

		cachePrefix := fmt.Sprintf("mapping_%d", index)
		cacheMiddleware := handler.cacheMiddlewareFactory(cachePrefix)
		setDefaultHandler(router, cacheMiddleware.Wrap(proxyHandler))
	}

	setDefaultHandler(handler.router, contracts.HandlerFunc(func(writer contracts.ResponseWriter, _ *http.Request) {
		infra.HTTPError(writer, errHostNotMapped)
	}))

	return handler
}

func (h *RequestHandler) ServeHTTP(writer contracts.ResponseWriter, request *contracts.Request) {
	h.router.ServeHTTP(writer, request)
}

func (h *RequestHandler) createHandler(response config.Response) http.Handler {
	return contracts.CastToHTTPHandler(
		mock.NewMockHandler(
			mock.WithLogger(ui.MockLogger),
			mock.WithResponse(response),
			mock.WithFileSystem(h.fs),
			mock.WithAfter(time.After),
		),
	)
}

func setDefaultHandler(router *mux.Router, handler contracts.Handler) {
	router.NotFoundHandler = contracts.CastToHTTPHandler(handler)
	router.MethodNotAllowedHandler = contracts.CastToHTTPHandler(handler)
}

const wildcard = "*"

func replaceWildcards(host string) string {
	count := strings.Count(host, wildcard)
	for i := 1; i <= count; i++ {
		host = strings.Replace(host, wildcard, sfmt.Sprintf("{p%d}", i), 1)
	}

	return host
}
