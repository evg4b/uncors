package script

import (
	"net/http"

	lua "github.com/yuin/gopher-lua"

	"github.com/evg4b/uncors/internal/contracts"
)

const (
	luaArgKey    = 2
	luaArgValue  = 3
	luaReturnOne = 1
	luaReturnTwo = 2
)

func createResponseTable(luaState *lua.LState, writer contracts.ResponseWriter) *lua.LTable {
	respTable := luaState.NewTable()
	headerWritten := false

	setupResponseMetatable(luaState, respTable)

	headersTable := createResponseHeadersTable(luaState, writer)
	respTable.RawSetString("headers", headersTable)

	addResponseMethods(luaState, respTable, writer, &headerWritten, headersTable)

	return respTable
}

func setupResponseMetatable(luaState *lua.LState, respTable *lua.LTable) {
	metatable := luaState.NewTable()

	indexFunc := luaState.NewFunction(func(state *lua.LState) int {
		key := state.CheckString(luaArgKey)

		switch key {
		case "status", "body":
			state.Push(lua.LNil)
		default:
			state.Push(respTable.RawGetString(key))
		}

		return luaReturnOne
	})
	metatable.RawSetString("__index", indexFunc)

	newindexFunc := luaState.NewFunction(func(state *lua.LState) int {
		key := state.CheckString(luaArgKey)
		value := state.Get(luaArgValue)

		switch key {
		case "status", "body":
			return 0
		default:
			respTable.RawSetString(key, value)
		}

		return 0
	})
	metatable.RawSetString("__newindex", newindexFunc)

	luaState.SetMetatable(respTable, metatable)
}

func createResponseHeadersTable(luaState *lua.LState, writer contracts.ResponseWriter) *lua.LTable {
	headersTable := luaState.NewTable()
	headersMetatable := luaState.NewTable()

	headersIndexFunc := luaState.NewFunction(func(state *lua.LState) int {
		key := state.CheckString(luaArgKey)

		if method := headersTable.RawGetString(key); method != lua.LNil {
			state.Push(method)

			return luaReturnOne
		}

		value := writer.Header().Get(key)
		state.Push(lua.LString(value))

		return luaReturnOne
	})
	headersMetatable.RawSetString("__index", headersIndexFunc)

	headersNewindexFunc := luaState.NewFunction(func(state *lua.LState) int {
		key := state.CheckString(luaArgKey)
		value := state.Get(luaArgValue)

		if value.Type() == lua.LTString {
			writer.Header().Set(key, value.String())
		}

		return 0
	})
	headersMetatable.RawSetString("__newindex", headersNewindexFunc)

	luaState.SetMetatable(headersTable, headersMetatable)

	addHeaderMethods(luaState, headersTable, writer)

	return headersTable
}

func addHeaderMethods(luaState *lua.LState, headersTable *lua.LTable, writer contracts.ResponseWriter) {
	setHeaderMethod := luaState.NewFunction(func(state *lua.LState) int {
		key := state.CheckString(luaArgKey)
		value := state.CheckString(luaArgValue)
		writer.Header().Set(key, value)

		return 0
	})
	headersTable.RawSetString("Set", setHeaderMethod)

	getHeaderMethod := luaState.NewFunction(func(state *lua.LState) int {
		key := state.CheckString(luaArgKey)
		value := writer.Header().Get(key)
		state.Push(lua.LString(value))

		return luaReturnOne
	})
	headersTable.RawSetString("Get", getHeaderMethod)
}

func addResponseMethods(
	luaState *lua.LState,
	respTable *lua.LTable,
	writer contracts.ResponseWriter,
	headerWritten *bool,
	headersTable *lua.LTable,
) {
	headerMethod := luaState.NewFunction(func(state *lua.LState) int {
		state.Push(headersTable)

		return luaReturnOne
	})
	respTable.RawSetString("Header", headerMethod)

	writeMethod := luaState.NewFunction(func(state *lua.LState) int {
		data := state.CheckString(luaArgKey)

		if !*headerWritten {
			writer.WriteHeader(http.StatusOK)
			*headerWritten = true
		}

		bytesWritten, err := writer.Write([]byte(data))
		if err != nil {
			state.Push(lua.LNumber(0))
			state.Push(lua.LString(err.Error()))

			return luaReturnTwo
		}

		state.Push(lua.LNumber(bytesWritten))
		state.Push(lua.LNil)

		return luaReturnTwo
	})
	respTable.RawSetString("Write", writeMethod)

	writeStringMethod := luaState.NewFunction(func(state *lua.LState) int {
		str := state.CheckString(luaArgKey)

		if !*headerWritten {
			writer.WriteHeader(http.StatusOK)
			*headerWritten = true
		}

		bytesWritten, err := writer.Write([]byte(str))
		if err != nil {
			state.Push(lua.LNumber(0))
			state.Push(lua.LString(err.Error()))

			return luaReturnTwo
		}

		state.Push(lua.LNumber(bytesWritten))
		state.Push(lua.LNil)

		return luaReturnTwo
	})
	respTable.RawSetString("WriteString", writeStringMethod)

	writeHeaderMethod := luaState.NewFunction(func(state *lua.LState) int {
		code := state.CheckInt(luaArgKey)

		if !*headerWritten {
			writer.WriteHeader(code)
			*headerWritten = true
		}

		return 0
	})
	respTable.RawSetString("WriteHeader", writeHeaderMethod)
}
