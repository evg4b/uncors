package router

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/infra"

	"github.com/gorilla/mux"
)

// maxRewriteRedispatch bounds how many times a request may be rewritten before
// it is refused, so that mutually recursive rewrite rules terminate.
const maxRewriteRedispatch = 8

var (
	errHostNotMapped = errors.New("host is not mapped in the uncors config")
	errRewriteLoop   = errors.New("rewrite rules redirect the request in a loop")
)

type rewriteDepthKey struct{}

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

	setDefaultHandler(instance.Router, infra.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) error {
		slog.Warn("host is not mapped", "host", request.Host)

		return infra.NewHTTPStatusError(
			http.StatusNotFound,
			"this host is not mapped in the uncors configuration",
			errHostNotMapped,
		)
	}))

	return &instance, nil
}

// registerMapping builds the request graph of a single mapping. The mapping gets
// a router of its own so that its cross-cutting middleware wraps every route it
// contains — mocks, scripts, statics and the proxy fallback alike — rather than
// decorating one branch and silently skipping the others.
func (r *Router) registerMapping(mapping config.Mapping) {
	mappingRouter := mux.NewRouter()
	routes := mappingRouter.Host(mapping.From.Hostname).Subrouter()

	// The cache stays on the upstream branch on purpose: caching a mock, a
	// script or a static file would only serve a stale copy of a response that
	// is already produced locally.
	upstream := r.deps.Proxy
	if len(mapping.Cache) > 0 {
		upstream = r.deps.Cache(mapping.Cache)(upstream)
	}

	// Registration order is the precedence policy: specific matchers first,
	// path prefix catch-alls (statics, commonly mounted at "/") last, and the
	// proxy fallback after everything else.
	registerMatchedRoutes(mapping.Mocks,
		func(m *config.Mock) *config.RequestMatcher { return &m.Matcher },
		func(def *config.Mock) {
			registerRoute(createRoute(routes, def.Matcher), r.deps.Mock(&def.Response))
		})

	registerMatchedRoutes(mapping.Scripts,
		func(s *config.Script) *config.RequestMatcher { return &s.Matcher },
		func(def *config.Script) {
			registerRoute(createRoute(routes, def.Matcher), r.deps.Script(def))
		})

	// A rewritten request re-enters the mapping's routes, so that it can be
	// answered by a mock, a script or a static file instead of always going
	// upstream.
	redispatch := redispatchHandler(mappingRouter)

	for _, rewriting := range mapping.Rewrites {
		registerPathHandler(routes, rewriting.From, r.deps.Rewrite(&rewriting)(redispatch))
	}

	for _, staticDir := range mapping.Statics {
		registerPrefixHandler(routes, staticDir.Path, r.deps.Static(staticDir.Path, staticDir)(upstream))
	}

	// The fallback is a route rather than a NotFoundHandler because gorilla/mux
	// does not run route middleware for the latter.
	registerRoute(routes.NewRoute(), upstream)

	r.Router.Host(mapping.From.Hostname).
		Handler(r.wrapMapping(mapping, mappingRouter))
}

// wrapMapping applies the middleware that belongs to the whole mapping rather
// than to one of its routes.
func (r *Router) wrapMapping(mapping config.Mapping, handler http.Handler) http.Handler {
	// A CORS preflight is a transport concern: it is answered before the router
	// decides whether the path is a mock, a static file or a proxy target.
	if !mapping.OptionsHandling.Disabled {
		handler = r.deps.Options(mapping.OptionsHandling)(handler)
	}

	// HAR records everything the mapping serves, locally produced responses
	// included, which is what "records the traffic of a mapping" means.
	if mapping.HAR.Enabled() {
		handler = r.deps.HAR(&mapping.HAR)(handler)
	}

	return handler
}

func redispatchHandler(routes http.Handler) http.Handler {
	return infra.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) error {
		depth, _ := request.Context().Value(rewriteDepthKey{}).(int)
		if depth >= maxRewriteRedispatch {
			return infra.NewHTTPStatusError(
				http.StatusLoopDetected,
				"the configured rewrites redirect this request in a loop",
				errRewriteLoop,
			)
		}

		ctx := context.WithValue(request.Context(), rewriteDepthKey{}, depth+1)
		routes.ServeHTTP(writer, request.WithContext(ctx))

		return nil
	})
}

func setDefaultHandler(router *mux.Router, handler http.Handler) {
	router.NotFoundHandler = handler
	router.MethodNotAllowedHandler = handler
}
