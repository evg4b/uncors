package script

import (
	"net/http"

	lua "github.com/yuin/gopher-lua"

	"github.com/evg4b/uncors/internal/contracts"
)

const (
	// Lua stack positions for metamethod arguments.
	luaArgKey   = 2 // Position of the key argument in metamethods
	luaArgValue = 3 // Position of the value argument in metamethods

	// Lua return values.
	luaReturnOne = 1 // Return one value from Lua function
	luaReturnTwo = 2 // Return two values from Lua function
)

// createResponseTable builds a Lua table representing the HTTP response.
// It provides methods for writing headers, status, and body directly to the ResponseWriter.
func createResponseTable(luaState *lua.LState, writer contracts.ResponseWriter) *lua.LTable {
	respTable := luaState.NewTable()
	headerWritten := false

	// Set up metatable to intercept property access
	setupResponseMetatable(luaState, respTable)

	// Create and configure headers table
	headersTable := createResponseHeadersTable(luaState, writer)
	respTable.RawSetString("headers", headersTable)

	// Add response methods
	addResponseMethods(luaState, respTable, writer, &headerWritten, headersTable)

	return respTable
}

// setupResponseMetatable configures the metatable to prevent direct access to status and body properties.
func setupResponseMetatable(luaState *lua.LState, respTable *lua.LTable) {
	metatable := luaState.NewTable()

	// __index: prevent reading status and body
	indexFunc := luaState.NewFunction(func(state *lua.LState) int {
		key := state.CheckString(luaArgKey)

		switch key {
		case "status", "body":
			// These properties don't exist - use methods instead
			state.Push(lua.LNil)
		default:
			state.Push(respTable.RawGetString(key))
		}

		return luaReturnOne
	})
	metatable.RawSetString("__index", indexFunc)

	// __newindex: prevent writing to status and body
	newindexFunc := luaState.NewFunction(func(state *lua.LState) int {
		key := state.CheckString(luaArgKey)
		value := state.Get(luaArgValue)

		switch key {
		case "status", "body":
			// Prevent direct assignment - use methods instead
			return 0
		default:
			respTable.RawSetString(key, value)
		}

		return 0
	})
	metatable.RawSetString("__newindex", newindexFunc)

	luaState.SetMetatable(respTable, metatable)
}

// createResponseHeadersTable creates a Lua table for response headers with direct write access.
func createResponseHeadersTable(luaState *lua.LState, writer contracts.ResponseWriter) *lua.LTable {
	headersTable := luaState.NewTable()

	// Create metatable for headers
	headersMetatable := luaState.NewTable()

	// __index: read headers from writer or return methods
	headersIndexFunc := luaState.NewFunction(func(state *lua.LState) int {
		key := state.CheckString(luaArgKey)

		// Check if it's a method first
		if method := headersTable.RawGetString(key); method != lua.LNil {
			state.Push(method)

			return luaReturnOne
		}

		// Otherwise, read from writer headers
		value := writer.Header().Get(key)
		state.Push(lua.LString(value))

		return luaReturnOne
	})
	headersMetatable.RawSetString("__index", headersIndexFunc)

	// __newindex: write headers to writer
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

	// Add header methods
	addHeaderMethods(luaState, headersTable, writer)

	return headersTable
}

// addHeaderMethods adds Get and Set methods to the headers table.
func addHeaderMethods(luaState *lua.LState, headersTable *lua.LTable, writer contracts.ResponseWriter) {
	// Set(key, value) method
	setHeaderMethod := luaState.NewFunction(func(state *lua.LState) int {
		key := state.CheckString(luaArgKey)
		value := state.CheckString(luaArgValue)
		writer.Header().Set(key, value)

		return 0
	})
	headersTable.RawSetString("Set", setHeaderMethod)

	// Get(key) method
	getHeaderMethod := luaState.NewFunction(func(state *lua.LState) int {
		key := state.CheckString(luaArgKey)
		value := writer.Header().Get(key)
		state.Push(lua.LString(value))

		return luaReturnOne
	})
	headersTable.RawSetString("Get", getHeaderMethod)
}

// addResponseMethods adds Write, WriteString, WriteHeader, and Header methods to the response table.
func addResponseMethods(
	luaState *lua.LState,
	respTable *lua.LTable,
	writer contracts.ResponseWriter,
	headerWritten *bool,
	headersTable *lua.LTable,
) {
	// Header() method - returns headers table
	headerMethod := luaState.NewFunction(func(state *lua.LState) int {
		state.Push(headersTable)

		return luaReturnOne
	})
	respTable.RawSetString("Header", headerMethod)

	// Write(data) method
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

	// WriteString(str) method
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

	// WriteHeader(statusCode) method
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
