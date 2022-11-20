package mock

import (
	"github.com/gorilla/mux"
)

func (m *Middleware) makeMockedRoutes() {
	var defaultMocks []Mock

	for _, mock := range m.mocks {
		if len(mock.Queries) > 0 || len(mock.Headers) > 0 || len(mock.Method) > 0 {
			route := m.router.NewRoute()
			setPath(route, mock.Path)
			setMethod(route, mock.Method)
			setQueries(route, mock.Queries)
			setHeaders(route, mock.Headers)
			route.Handler(m.makeHandler(mock.Response))
		} else {
			defaultMocks = append(defaultMocks, mock)
		}
	}

	for _, mock := range defaultMocks {
		route := m.router.NewRoute()
		setPath(route, mock.Path)
		route.Handler(m.makeHandler(mock.Response))
	}
}

func (m *Middleware) makeHandler(response Response) *internalHandler {
	return &internalHandler{response, m.logger, m.fs}
}

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
