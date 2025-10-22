package handler

import (
	"github.com/evg4b/uncors/internal/config"
	"github.com/gorilla/mux"
)

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
