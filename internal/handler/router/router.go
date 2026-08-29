package router

import (
	"errors"
	"net/http"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/infra"

	"github.com/gorilla/mux"
)

var errHostNotMapped = errors.New("host not mapped")

func setDefaultHandler(router *mux.Router, handler http.Handler) {
	router.NotFoundHandler = handler
	router.MethodNotAllowedHandler = handler
}

type Router struct {
	*mux.Router

	deps Deps
}

func NewRouter(mappings config.Mappings, deps Deps) (*Router, error) {
	instance := Router{
		Router: mux.NewRouter(),
		deps:   deps,
	}

	for _, mapping := range mappings {
		instance.registerMapping(mapping)
	}

	setDefaultHandler(instance.Router, infra.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) error {
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
		middleware := r.deps.Static(staticDir.Path, staticDir)
		registerPrefixHandler(router, staticDir.Path, middleware(defaultHandler))
	}

	registerMatchedRoutes(mapping.Mocks,
		func(m *config.Mock) *config.RequestMatcher { return &m.Matcher },
		func(def *config.Mock) {
			registerRoute(createRoute(router, def.Matcher), r.deps.Mock(&def.Response))
		})

	registerMatchedRoutes(mapping.Scripts,
		func(s *config.Script) *config.RequestMatcher { return &s.Matcher },
		func(def *config.Script) {
			registerRoute(createRoute(router, def.Matcher), r.deps.Script(def))
		})

	for _, rewrite := range mapping.Rewrites {
		wrappedHandler := r.deps.Rewrite(&rewrite)(defaultHandler)

		registerPathHandler(router, rewrite.From, wrappedHandler)
	}

	setDefaultHandler(router, defaultHandler)
}

func (r *Router) prepareDefaultHandler(mapping config.Mapping) http.Handler {
	defaultHandler := r.deps.Proxy
	if !mapping.OptionsHandling.Disabled {
		defaultHandler = r.deps.Options(mapping.OptionsHandling)(defaultHandler)
	}

	if len(mapping.Cache) > 0 {
		defaultHandler = r.deps.Cache(mapping.Cache)(defaultHandler)
	}

	if mapping.HAR.Enabled() {
		defaultHandler = r.deps.HAR(&mapping.HAR)(defaultHandler)
	}

	return defaultHandler
}
