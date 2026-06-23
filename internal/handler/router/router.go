package router

import (
	"errors"
	"net/http"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/infra"

	"github.com/gorilla/mux"
)

var errHostNotMapped = errors.New("host not mapped")

func setDefaultHandler(router *mux.Router, handler contracts.Handler) {
	httpHandler := infra.CastToHTTPHandler(handler)
	router.NotFoundHandler = httpHandler
	router.MethodNotAllowedHandler = httpHandler
}

type DI interface {
	StaticMiddleware(path string, dir config.StaticDirectory) contracts.Middleware
	RewriteMiddleware(rewriting *config.RewritingOption) contracts.Middleware
	HARMiddleware(harConfig *config.HARConfig) contracts.Middleware
	ScriptHandler(scriptConfig *config.Script) contracts.Handler
	OptionsMiddleware(cfg config.OptionsHandling) contracts.Middleware
	MockHandler(response *config.Response) contracts.Handler
}

type Router struct {
	*mux.Router

	defaultHandler contracts.Handler
	container      DI

	cacheMiddlewareFactory CacheMiddlewareFactory
}

func NewRouter(mappings config.Mappings, options ...Option) (*Router, error) {
	instance := Router{
		Router: mux.NewRouter(),
	}

	for _, option := range options {
		option(&instance)
	}

	for _, mapping := range mappings {
		instance.registerMapping(mapping)
	}

	setDefaultHandler(instance.Router, infra.HandlerFunc(func(_ contracts.ResponseWriter, _ *http.Request) error {
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
		middleware := r.container.StaticMiddleware(staticDir.Path, staticDir)
		registerPrefixHandler(router, staticDir.Path, infra.Mddleware(middleware, defaultHandler))
	}

	registerMatchedRoutes(mapping.Mocks,
		func(m *config.Mock) *config.RequestMatcher { return &m.Matcher },
		func(def *config.Mock) {
			registerRoute(createRoute(router, def.Matcher), r.container.MockHandler(&def.Response))
		})

	registerMatchedRoutes(mapping.Scripts,
		func(s *config.Script) *config.RequestMatcher { return &s.Matcher },
		func(def *config.Script) {
			registerRoute(createRoute(router, def.Matcher), r.container.ScriptHandler(def))
		})

	for _, rewrite := range mapping.Rewrites {
		wrappedHandler := infra.Mddleware(r.container.RewriteMiddleware(&rewrite), defaultHandler)

		registerPathHandler(router, rewrite.From, wrappedHandler)
	}

	setDefaultHandler(router, defaultHandler)
}

func (r *Router) prepareDefaultHandler(mapping config.Mapping) contracts.Handler {
	defaultHandler := r.defaultHandler
	if !mapping.OptionsHandling.Disabled {
		defaultHandler = infra.Mddleware(r.container.OptionsMiddleware(mapping.OptionsHandling), defaultHandler)
	}

	if len(mapping.Cache) > 0 {
		defaultHandler = infra.Mddleware(r.cacheMiddlewareFactory(mapping.Cache), defaultHandler)
	}

	if mapping.HAR.Enabled() {
		defaultHandler = infra.Mddleware(r.container.HARMiddleware(&mapping.HAR), defaultHandler)
	}

	return defaultHandler
}
