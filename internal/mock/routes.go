package mock

import "github.com/gorilla/mux"

func MakeMockedRoutes(router *mux.Router, mocks []Mock) {
	for _, mock := range mocks {
		route := router.NewRoute()
		if len(mock.Path) > 0 {
			route.Path(mock.Path)
		}

		if len(mock.Queries) > 0 {
			for key, value := range mock.Queries {
				route.Queries(key, value)
			}
		}

		if len(mock.Headers) > 0 {
			for key, value := range mock.Headers {
				route.Headers(key, value)
			}
		}

		route.Handler(&Handler{mock: mock})
	}
}
