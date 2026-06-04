package handler

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/gorilla/mux"
)

type (
	CacheMiddlewareFactory   = func(globs config.CacheGlobs) contracts.Middleware
	StaticMiddlewareFactory  = func(path string, dir config.StaticDirectory) contracts.Middleware
	MockHandlerFactory       = func(response config.Response) contracts.Handler
	ScriptHandlerFactory     = func(script config.Script) contracts.Handler
	RewriteMiddlewareFactory = func(rewrite config.RewritingOption) contracts.Middleware
	OptionsMiddlewareFactory = func(options config.OptionsHandling) contracts.Middleware
	HARMiddlewareFactory     = func(harConfig config.HARConfig) contracts.Middleware
)

type RequestHandler struct {
	*mux.Router

	mappings                 config.Mappings
	output                   contracts.Output
	cacheMiddlewareFactory   CacheMiddlewareFactory
	staticMiddlewareFactory  StaticMiddlewareFactory
	proxyHandler             contracts.Handler
	mockHandlerFactory       MockHandlerFactory
	scriptHandlerFactory     ScriptHandlerFactory
	rewriteMiddlewareFactory RewriteMiddlewareFactory
	optionsMiddlewareFactory OptionsMiddlewareFactory
	harMiddlewareFactory     HARMiddlewareFactory
}

// MiddlewareFunc adapts an ordinary func into a contracts.Middleware.
type MiddlewareFunc func(contracts.Handler) contracts.Handler

func (f MiddlewareFunc) Wrap(next contracts.Handler) contracts.Handler {
	return f(next)
}

var errHostNotMapped = errors.New("host not mapped")

func NewUncorsRequestHandler(options ...RequestHandlerOption) *RequestHandler {
	handler := helpers.ApplyOptions(&RequestHandler{Router: mux.NewRouter(), mappings: config.Mappings{}}, options)

	helpers.AssertIsDefined(handler.cacheMiddlewareFactory, "Cache middleware is not set")
	helpers.AssertIsDefined(handler.proxyHandler, "Proxy handler is not set")

	for _, mapping := range handler.mappings {
		host, _, err := mapping.GetFromHostPort()
		if err != nil {
			panic(err)
		}

		router := handler.Host(replaceWildcards(host)).Subrouter()

		defaultHandler := handler.wrapOptionsMiddleware(mapping.OptionsHandling, handler.proxyHandler)
		defaultHandler = handler.wrapCacheMiddleware(mapping.Cache, defaultHandler)
		defaultHandler = handler.wrapHARMiddleware(mapping.HAR, defaultHandler)

		handler.makeStaticRoutes(router, mapping.Statics, defaultHandler)
		handler.makeMockedRoutes(router, mapping.Mocks)
		handler.makeScriptRoutes(router, mapping.Scripts)
		handler.makeRewrittenRoutes(router, mapping.Rewrites, defaultHandler)

		setDefaultHandler(router, defaultHandler)
	}

	setDefaultHandler(handler.Router, contracts.HandlerFunc(func(_ contracts.ResponseWriter, r *http.Request) error {
		handler.output.Errorf("Host %s://%s is not mapped", r.URL.Scheme, r.URL.Host)
		log.Printf("Host %s://%s is not mapped", r.URL.Scheme, r.URL.Host) // nolint: gosec

		return errHostNotMapped
	}))

	return handler
}

func (h *RequestHandler) ServeHTTP(writer contracts.ResponseWriter, request *contracts.Request) error {
	h.Router.ServeHTTP(writer, request)

	return nil
}

func (h *RequestHandler) createHandler(response config.Response) http.Handler {
	return contracts.CastToHTTPHandler(
		h.mockHandlerFactory(response),
	)
}

func (h *RequestHandler) registerRoute(route *mux.Route, handler contracts.Handler) {
	route.Handler(contracts.CastToHTTPHandler(handler))
}

func setDefaultHandler(router *mux.Router, handler contracts.Handler) {
	httpHandler := contracts.CastToHTTPHandler(handler)
	router.NotFoundHandler = httpHandler
	router.MethodNotAllowedHandler = httpHandler
}

const wildcard = "*"

func replaceWildcards(host string) string {
	count := strings.Count(host, wildcard)
	for i := 1; i <= count; i++ {
		host = strings.Replace(host, wildcard, fmt.Sprintf("{p%d}", i), 1)
	}

	return host
}

type RequestHandlerOption = func(*RequestHandler)

func WithMappings(mappings config.Mappings) RequestHandlerOption {
	return func(h *RequestHandler) {
		h.mappings = mappings
	}
}

func WithOutput(output contracts.Output) RequestHandlerOption {
	return func(h *RequestHandler) {
		h.output = output
	}
}

func WithCacheMiddlewareFactory(factory CacheMiddlewareFactory) RequestHandlerOption {
	return func(h *RequestHandler) {
		h.cacheMiddlewareFactory = factory
	}
}

func WithProxyHandler(proxyHandler contracts.Handler) RequestHandlerOption {
	return func(h *RequestHandler) {
		h.proxyHandler = proxyHandler
	}
}

func WithStaticHandlerFactory(factory StaticMiddlewareFactory) RequestHandlerOption {
	return func(h *RequestHandler) {
		h.staticMiddlewareFactory = factory
	}
}

func WithMockHandlerFactory(factory MockHandlerFactory) RequestHandlerOption {
	return func(h *RequestHandler) {
		h.mockHandlerFactory = factory
	}
}

func WithScriptHandlerFactory(factory ScriptHandlerFactory) RequestHandlerOption {
	return func(h *RequestHandler) {
		h.scriptHandlerFactory = factory
	}
}

func WithRewriteHandlerFactory(factory RewriteMiddlewareFactory) RequestHandlerOption {
	return func(h *RequestHandler) {
		h.rewriteMiddlewareFactory = factory
	}
}

func WithOptionsHandlerFactory(factory OptionsMiddlewareFactory) RequestHandlerOption {
	return func(h *RequestHandler) {
		h.optionsMiddlewareFactory = factory
	}
}

func WithHARMiddlewareFactory(factory HARMiddlewareFactory) RequestHandlerOption {
	return func(h *RequestHandler) {
		h.harMiddlewareFactory = factory
	}
}
