package handler

import (
	"errors"
	"net/http"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/gorilla/mux"
)

var errHostNotMapped = errors.New("host not mapped")

func setDefaultHandler(router *mux.Router, handler contracts.Handler) {
	httpHandler := contracts.CastToHTTPHandler(handler)
	router.NotFoundHandler = httpHandler
	router.MethodNotAllowedHandler = httpHandler
}

type Router struct {
	*mux.Router

	defaultHandler contracts.Handler

	cacheMiddlewareFactory   CacheMiddlewareFactory
	staticMiddlewareFactory  StaticMiddlewareFactory
	mockHandlerFactory       MockHandlerFactory
	scriptHandlerFactory     ScriptHandlerFactory
	rewriteMiddlewareFactory RewriteMiddlewareFactory
	optionsMiddlewareFactory OptionsMiddlewareFactory
	harMiddlewareFactory     HARMiddlewareFactory
}

func NewRouter(mappings config.Mappings, options ...RouterOption) (*Router, error) {
	instance := Router{
		Router: mux.NewRouter(),
	}

	for _, option := range options {
		option(&instance)
	}

	for _, mapping := range mappings {
		instance.registerMapping(mapping)
	}

	setDefaultHandler(instance.Router, contracts.HandlerFunc(func(_ contracts.ResponseWriter, _ *http.Request) error {
		// instance.output.Errorf("Host %s://%s is not mapped", r.URL.Scheme, r.URL.Host)
		// log.Printf("Host %s://%s is not mapped", r.URL.Scheme, r.URL.Host) // nolint: gosec
		return errHostNotMapped
	}))

	return &instance, nil
}

func (r *Router) registerMapping(mapping config.Mapping) {
	router := r.Router.Host(mapping.From.Hostname).
		Subrouter()

	defaultHandler := r.prepareDefaultHandler(mapping)

	for _, staticDir := range mapping.Statics {
		middleware := r.staticMiddlewareFactory(staticDir.Path, staticDir)
		registerPrefixHandler(router, staticDir.Path, Mddleware(middleware, defaultHandler))
	}

	registerMatchedRoutes(mapping.Mocks,
		func(m *config.Mock) *config.RequestMatcher { return &m.Matcher },
		func(def *config.Mock) {
			registerRoute(createRoute(router, def.Matcher), r.mockHandlerFactory(def.Response))
		})

	registerMatchedRoutes(mapping.Scripts,
		func(s *config.Script) *config.RequestMatcher { return &s.Matcher },
		func(def *config.Script) {
			registerRoute(createRoute(router, def.Matcher), r.scriptHandlerFactory(*def))
		})

	for _, rewrite := range mapping.Rewrites {
		wrappedHandler := Mddleware(r.rewriteMiddlewareFactory(rewrite), defaultHandler)

		registerPathHandler(router, rewrite.From, wrappedHandler)
	}

	setDefaultHandler(router, defaultHandler)
}

func (r *Router) prepareDefaultHandler(mapping config.Mapping) contracts.Handler {
	defaultHandler := r.defaultHandler
	if !mapping.OptionsHandling.Disabled {
		defaultHandler = Mddleware(r.optionsMiddlewareFactory(mapping.OptionsHandling), defaultHandler)
	}

	if len(mapping.Cache) > 0 {
		defaultHandler = Mddleware(r.cacheMiddlewareFactory(mapping.Cache), defaultHandler)
	}

	if mapping.HAR.Enabled() {
		defaultHandler = Mddleware(r.harMiddlewareFactory(mapping.HAR), defaultHandler)
	}

	return defaultHandler
}
