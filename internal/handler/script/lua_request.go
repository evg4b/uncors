package script

import (
	"io"

	"github.com/gorilla/mux"
	lua "github.com/yuin/gopher-lua"

	"github.com/evg4b/uncors/internal/contracts"
)

func createRequestTable(luaState *lua.LState, request *contracts.Request) *lua.LTable {
	reqTable := luaState.NewTable()

	reqTable.RawSetString("method", lua.LString(request.Method))
	reqTable.RawSetString("url", lua.LString(request.URL.String()))
	reqTable.RawSetString("path", lua.LString(request.URL.Path))
	reqTable.RawSetString("query", lua.LString(request.URL.RawQuery))
	reqTable.RawSetString("host", lua.LString(request.Host))
	reqTable.RawSetString("remote_addr", lua.LString(request.RemoteAddr))

	reqTable.RawSetString("headers", createHeadersTable(luaState, request.Header))
	reqTable.RawSetString("query_params", createQueryParamsTable(luaState, request.URL.Query()))
	reqTable.RawSetString("path_params", createPathParamsTable(luaState, request))

	if request.Body != nil {
		if body, err := io.ReadAll(request.Body); err == nil {
			reqTable.RawSetString("body", lua.LString(string(body)))
		}
	}

	return reqTable
}

func createHeadersTable(luaState *lua.LState, headers map[string][]string) *lua.LTable {
	headersTable := luaState.NewTable()

	for key, values := range headers {
		if len(values) == 1 {
			headersTable.RawSetString(key, lua.LString(values[0]))
		} else {
			valuesList := luaState.NewTable()
			for _, value := range values {
				valuesList.Append(lua.LString(value))
			}
			headersTable.RawSetString(key, valuesList)
		}
	}

	return headersTable
}

func createQueryParamsTable(luaState *lua.LState, queryParams map[string][]string) *lua.LTable {
	queryTable := luaState.NewTable()

	for key, values := range queryParams {
		if len(values) == 1 {
			queryTable.RawSetString(key, lua.LString(values[0]))
		} else {
			valuesList := luaState.NewTable()
			for _, value := range values {
				valuesList.Append(lua.LString(value))
			}
			queryTable.RawSetString(key, valuesList)
		}
	}

	return queryTable
}

func createPathParamsTable(luaState *lua.LState, request *contracts.Request) *lua.LTable {
	pathVarsTable := luaState.NewTable()

	pathVars := mux.Vars(request)
	for key, value := range pathVars {
		pathVarsTable.RawSetString(key, lua.LString(value))
	}

	return pathVarsTable
}
