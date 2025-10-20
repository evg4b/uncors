package handler

import (
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/gorilla/mux"
)

func (h *RequestHandler) makeLuaScriptRoutes(router *mux.Router, scripts config.LuaScripts) {
	var defaultScripts config.LuaScripts

	for _, scriptDef := range scripts {
		if len(scriptDef.Queries) > 0 || len(scriptDef.Headers) > 0 || len(scriptDef.Method) > 0 {
			route := router.NewRoute()
			setPath(route, scriptDef.Path)
			setMethod(route, scriptDef.Method)
			setQueries(route, scriptDef.Queries)
			setHeaders(route, scriptDef.Headers)
			route.Handler(contracts.CastToHTTPHandler(h.luaHandlerFactory(scriptDef)))
		} else {
			defaultScripts = append(defaultScripts, scriptDef)
		}
	}

	for _, scriptDef := range defaultScripts {
		route := router.NewRoute()
		setPath(route, scriptDef.Path)
		route.Handler(contracts.CastToHTTPHandler(h.luaHandlerFactory(scriptDef)))
	}
}
