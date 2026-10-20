package router

import (
	"net/http"
	"strings"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/gorilla/mux"
)

func createRoute(router *mux.Router, matcher config.RequestMatcher) *mux.Route {
	route := router.NewRoute()

	if matcher.Path != "" {
		route.Path(matcher.Path)
	}

	if matcher.Method != "" {
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

func registerPathHandler(router *mux.Router, path string, handler contracts.Handler) {
	clearPath, fullPath := normalizePath(path)

	registerRoute(router.NewRoute().Path(clearPath), handler)
	registerRoute(router.NewRoute().PathPrefix(fullPath), handler)
}

func registerPrefixHandler(router *mux.Router, prefix string, handler contracts.Handler) {
	clearPrefix, fullPrefix := normalizePath(prefix)
	if clearPrefix == "" {
		registerPathHandler(router, prefix, handler)

		return
	}

	router.NewRoute().
		Path(clearPrefix).
		HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			target := fullPrefix
			if request.URL.RawQuery != "" {
				target += "?" + request.URL.RawQuery
			}

			http.Redirect(writer, request, target, http.StatusTemporaryRedirect)
		})

	registerRoute(router.NewRoute().PathPrefix(fullPrefix), handler)
}

func registerRoute(route *mux.Route, handler contracts.Handler) {
	route.Handler(infra.CastToHTTPHandler(handler))
}

func normalizePath(path string) (string, string) {
	clearPath := strings.TrimSuffix(path, "/")
	fullPath := clearPath + "/"

	return clearPath, fullPath
}
