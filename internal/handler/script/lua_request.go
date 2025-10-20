package script

import (
	"io"

	"github.com/gorilla/mux"
	lua "github.com/yuin/gopher-lua"

	"github.com/evg4b/uncors/internal/contracts"
)

// createRequestTable builds a Lua table representing the HTTP request.
// It includes method, URL, headers, query parameters, path parameters, and body.
func createRequestTable(L *lua.LState, request *contracts.Request) *lua.LTable {
	reqTable := L.NewTable()

	// Basic request properties
	reqTable.RawSetString("method", lua.LString(request.Method))
	reqTable.RawSetString("url", lua.LString(request.URL.String()))
	reqTable.RawSetString("path", lua.LString(request.URL.Path))
	reqTable.RawSetString("query", lua.LString(request.URL.RawQuery))
	reqTable.RawSetString("host", lua.LString(request.Host))
	reqTable.RawSetString("remote_addr", lua.LString(request.RemoteAddr))

	// Headers
	reqTable.RawSetString("headers", createHeadersTable(L, request.Header))

	// Query parameters
	reqTable.RawSetString("query_params", createQueryParamsTable(L, request.URL.Query()))

	// Path parameters (from gorilla/mux)
	reqTable.RawSetString("path_params", createPathParamsTable(L, request))

	// Request body
	if request.Body != nil {
		if body, err := io.ReadAll(request.Body); err == nil {
			reqTable.RawSetString("body", lua.LString(string(body)))
		}
	}

	return reqTable
}

// createHeadersTable converts HTTP headers to a Lua table.
// Single-value headers are stored as strings, multi-value headers as tables.
func createHeadersTable(L *lua.LState, headers map[string][]string) *lua.LTable {
	headersTable := L.NewTable()

	for key, values := range headers {
		if len(values) == 1 {
			headersTable.RawSetString(key, lua.LString(values[0]))
		} else {
			valuesList := L.NewTable()
			for _, v := range values {
				valuesList.Append(lua.LString(v))
			}
			headersTable.RawSetString(key, valuesList)
		}
	}

	return headersTable
}

// createQueryParamsTable converts URL query parameters to a Lua table.
// Single-value params are stored as strings, multi-value params as tables.
func createQueryParamsTable(L *lua.LState, queryParams map[string][]string) *lua.LTable {
	queryTable := L.NewTable()

	for key, values := range queryParams {
		if len(values) == 1 {
			queryTable.RawSetString(key, lua.LString(values[0]))
		} else {
			valuesList := L.NewTable()
			for _, v := range values {
				valuesList.Append(lua.LString(v))
			}
			queryTable.RawSetString(key, valuesList)
		}
	}

	return queryTable
}

// createPathParamsTable extracts path parameters from gorilla/mux and converts them to a Lua table.
func createPathParamsTable(L *lua.LState, request *contracts.Request) *lua.LTable {
	pathVarsTable := L.NewTable()

	pathVars := mux.Vars(request)
	for key, value := range pathVars {
		pathVarsTable.RawSetString(key, lua.LString(value))
	}

	return pathVarsTable
}
