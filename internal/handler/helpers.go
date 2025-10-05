package handler

import (
	"github.com/gorilla/mux"
)

func setPath(route *mux.Route, path string) {
	if len(path) > 0 {
		route.Path(path)
	}
}

func setMethod(route *mux.Route, methods string) {
	if len(methods) > 0 {
		route.Methods(methods)
	}
}

func setQueries(route *mux.Route, queries map[string]string) {
	if len(queries) > 0 {
		for key, value := range queries {
			route.Queries(key, value)
		}
	}
}

func setHeaders(route *mux.Route, headers map[string]string) {
	if len(headers) > 0 {
		for key, value := range headers {
			route.Headers(key, value)
		}
	}
}
