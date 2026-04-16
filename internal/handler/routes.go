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
	var defaultMocks config.Mocks

	for _, mockDef := range mocks {
		if !mockDef.Matcher.IsPathOnly() {
			h.createRoute(router, mockDef.Matcher).
				Handler(h.createHandler(mockDef.Response))
		} else {
			defaultMocks = append(defaultMocks, mockDef)
		}
	}

	for _, mockDef := range defaultMocks {
		h.createRoute(router, mockDef.Matcher).
			Handler(h.createHandler(mockDef.Response))
	}
}

func (h *RequestHandler) makeScriptRoutes(router *mux.Router, scripts config.Scripts) {
	var defaultScripts config.Scripts

	for _, scriptDef := range scripts {
		if !scriptDef.Matcher.IsPathOnly() {
			h.createRoute(router, scriptDef.Matcher).
				Handler(contracts.CastToHTTPHandler(h.scriptHandlerFactory(scriptDef)))
		} else {
			defaultScripts = append(defaultScripts, scriptDef)
		}
	}

	for _, scriptDef := range defaultScripts {
		h.createRoute(router, scriptDef.Matcher).
			Handler(contracts.CastToHTTPHandler(h.scriptHandlerFactory(scriptDef)))
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
