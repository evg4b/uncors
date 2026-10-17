package script

import (
	"fmt"
	"io"

	"github.com/gorilla/mux"
	lua "github.com/yuin/gopher-lua"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/pkg/urlt"
)

const DefaultMaxBodySize int64 = 10 << 20

func createRequestTable(luaState *lua.LState, request *contracts.Request, maxBodySize int64) (*lua.LTable, error) {
	reqTable := luaState.NewTable()

	reqTable.RawSetString("method", lua.LString(request.Method))
	reqTable.RawSetString("url", lua.LString(urlt.URL_String(request.URL)))
	reqTable.RawSetString("path", lua.LString(request.URL.Path))
	reqTable.RawSetString("query", lua.LString(request.URL.RawQuery))
	reqTable.RawSetString("host", lua.LString(request.Host))
	reqTable.RawSetString("remote_addr", lua.LString(request.RemoteAddr))

	reqTable.RawSetString("headers", createValuesTable(luaState, request.Header))
	reqTable.RawSetString("query_params", createValuesTable(luaState, urlt.URL_Query(request.URL)))
	reqTable.RawSetString("path_params", createPathParamsTable(luaState, request))

	body, err := readBody(request, maxBodySize)
	if err != nil {
		return nil, err
	}

	reqTable.RawSetString("body", lua.LString(body))

	return reqTable, nil
}

func readBody(request *contracts.Request, maxBodySize int64) (string, error) {
	if request.Body == nil {
		return "", nil
	}

	body, err := io.ReadAll(io.LimitReader(request.Body, maxBodySize+1))
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrRequestBodyNotRead, err)
	}

	if int64(len(body)) > maxBodySize {
		return "", fmt.Errorf("%w: limit is %d bytes", ErrRequestBodyTooLarge, maxBodySize)
	}

	return string(body), nil
}

func createValuesTable(luaState *lua.LState, values map[string][]string) *lua.LTable {
	table := luaState.NewTable()

	for key, items := range values {
		if len(items) == 1 {
			table.RawSetString(key, lua.LString(items[0]))

			continue
		}

		list := luaState.NewTable()
		for _, item := range items {
			list.Append(lua.LString(item))
		}

		table.RawSetString(key, list)
	}

	return table
}

func createPathParamsTable(luaState *lua.LState, request *contracts.Request) *lua.LTable {
	pathVarsTable := luaState.NewTable()

	pathVars := mux.Vars(request)
	for key, value := range pathVars {
		pathVarsTable.RawSetString(key, lua.LString(value))
	}

	return pathVarsTable
}
