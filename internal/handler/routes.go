package handler

import (
	"net/http"
	"strings"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/gorilla/mux"
)

func (h *RequestHandler) makeStaticRoutes(
	router *mux.Router,
	statics config.StaticDirectories,
	next contracts.Handler,
) {
	for _, staticDir := range statics {
		clearPath := strings.TrimSuffix(staticDir.Path, "/")
		path := clearPath + "/"

		router.NewRoute().
			Path(clearPath).
			Handler(http.RedirectHandler(path, http.StatusTemporaryRedirect))

		middleware := h.staticMiddlewareFactory(path, staticDir)
		router.NewRoute().
			PathPrefix(path).
			Handler(contracts.CastToHTTPHandler(middleware.Wrap(next)))
	}
}

func (h *RequestHandler) makeMockedRoutes(router *mux.Router, mocks config.Mocks) {
	matcher := func(def config.Mock) *config.RequestMatcher { return &def.Matcher }
	register := func(def config.Mock) {
		h.createRoute(router, def.Matcher).Handler(h.createHandler(def.Response))
	}
	registerMatchedRoutes(mocks, matcher, register)
}

func (h *RequestHandler) makeScriptRoutes(router *mux.Router, scripts config.Scripts) {
	matcher := func(def config.Script) *config.RequestMatcher { return &def.Matcher }
	register := func(def config.Script) {
		h.createRoute(router, def.Matcher).Handler(contracts.CastToHTTPHandler(h.scriptHandlerFactory(def)))
	}
	registerMatchedRoutes(scripts, matcher, register)
}

// registerMatchedRoutes registers routes in two passes: specific matchers first, path-only matchers second.
// This ensures specific routes take priority over catch-all path routes in gorilla/mux.
func registerMatchedRoutes[T any](items []T, matcher func(T) *config.RequestMatcher, register func(T)) {
	var defaults []T

	for _, item := range items {
		if !matcher(item).IsPathOnly() {
			register(item)
		} else {
			defaults = append(defaults, item)
		}
	}

	for _, item := range defaults {
		register(item)
	}
}

func (h *RequestHandler) makeRewrittenRoutes(
	router *mux.Router,
	rewrites config.RewriteOptions,
	next contracts.Handler,
) {
	for _, rewrite := range rewrites {
		clearPath := strings.TrimSuffix(rewrite.From, "/")
		path := clearPath + "/"

		middleware := h.rewriteMiddlewareFactory(rewrite)
		handler := contracts.CastToHTTPHandler(middleware.Wrap(next))

		router.NewRoute().Path(clearPath).Handler(handler)
		router.NewRoute().PathPrefix(path).Handler(handler)
	}
}

func (h *RequestHandler) wrapCacheMiddleware(cache config.CacheGlobs, next contracts.Handler) contracts.Handler {
	if len(cache) > 0 {
		return h.cacheMiddlewareFactory(cache).Wrap(next)
	}

	return next
}

func (h *RequestHandler) wrapOptionsMiddleware(opts config.OptionsHandling, next contracts.Handler) contracts.Handler {
	if opts.Disabled {
		return next
	}

	return h.optionsMiddlewareFactory(opts).Wrap(next)
}

func (h *RequestHandler) createRoute(router *mux.Router, matcher config.RequestMatcher) *mux.Route {
	route := router.NewRoute()

	if len(matcher.Path) > 0 {
		route.Path(matcher.Path)
	}

	if len(matcher.Method) > 0 {
		route.Methods(matcher.Method)
	}

	for key, value := range matcher.Queries {
		route.Queries(key, value)
	}

	for key, value := range matcher.Headers {
		route.Headers(key, value)
	}

	return route
}
