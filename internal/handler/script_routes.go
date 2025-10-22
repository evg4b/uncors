package handler

import (
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/gorilla/mux"
)

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
