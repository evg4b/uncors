package handler

import (
	"github.com/evg4b/uncors/internal/config"
	"github.com/gorilla/mux"
)

func (h *RequestHandler) createRoute(router *mux.Router, matcher config.RequestMatcher) *mux.Route {
	route := router.NewRoute()
	if len(matcher.Path) > 0 {
		route.Path(matcher.Path)
	}

	if len(matcher.Method) > 0 {
		route.Methods(matcher.Method)
	}

	if len(matcher.Queries) > 0 {
		for key, value := range matcher.Queries {
			route.Queries(key, value)
		}
	}

	if len(matcher.Headers) > 0 {
		for key, value := range matcher.Headers {
			route.Headers(key, value)
		}
	}

	return route
}
