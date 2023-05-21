package handler

import (
	"github.com/evg4b/uncors/internal/config"
)

func (m *UncorsRequestHandler) makeMockedRoutes() {
	var defaultMocks []config.Mock

	for _, mockDef := range m.mocks {
		if len(mockDef.Queries) > 0 || len(mockDef.Headers) > 0 || len(mockDef.Method) > 0 {
			route := m.router.NewRoute()
			setPath(route, mockDef.Path)
			setMethod(route, mockDef.Method)
			setQueries(route, mockDef.Queries)
			setHeaders(route, mockDef.Headers)
			route.Handler(m.createHandler(mockDef.Response))
		} else {
			defaultMocks = append(defaultMocks, mockDef)
		}
	}

	for _, mockDef := range defaultMocks {
		route := m.router.NewRoute()
		setPath(route, mockDef.Path)
		route.Handler(m.createHandler(mockDef.Response))
	}
}
