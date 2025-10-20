package script

import (
	"net/http"

	lua "github.com/yuin/gopher-lua"

	"github.com/evg4b/uncors/internal/contracts"
)

// createResponseTable builds a Lua table representing the HTTP response.
// It provides methods for writing headers, status, and body directly to the ResponseWriter.
func createResponseTable(L *lua.LState, writer contracts.ResponseWriter) *lua.LTable {
	respTable := L.NewTable()
	headerWritten := false

	// Set up metatable to intercept property access
	setupResponseMetatable(L, respTable)

	// Create and configure headers table
	headersTable := createResponseHeadersTable(L, writer)
	respTable.RawSetString("headers", headersTable)

	// Add response methods
	addResponseMethods(L, respTable, writer, &headerWritten, headersTable)

	return respTable
}

// setupResponseMetatable configures the metatable to prevent direct access to status and body properties.
func setupResponseMetatable(L *lua.LState, respTable *lua.LTable) {
	metatable := L.NewTable()

	// __index: prevent reading status and body
	indexFunc := L.NewFunction(func(L *lua.LState) int {
		key := L.CheckString(2)

		switch key {
		case "status", "body":
			// These properties don't exist - use methods instead
			L.Push(lua.LNil)
		default:
			L.Push(respTable.RawGetString(key))
		}

		return 1
	})
	metatable.RawSetString("__index", indexFunc)

	// __newindex: prevent writing to status and body
	newindexFunc := L.NewFunction(func(L *lua.LState) int {
		key := L.CheckString(2)
		value := L.Get(3)

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

	L.SetMetatable(respTable, metatable)
}

// createResponseHeadersTable creates a Lua table for response headers with direct write access.
func createResponseHeadersTable(L *lua.LState, writer contracts.ResponseWriter) *lua.LTable {
	headersTable := L.NewTable()

	// Create metatable for headers
	headersMetatable := L.NewTable()

	// __index: read headers from writer or return methods
	headersIndexFunc := L.NewFunction(func(L *lua.LState) int {
		key := L.CheckString(2)

		// Check if it's a method first
		if method := headersTable.RawGetString(key); method != lua.LNil {
			L.Push(method)
			return 1
		}

		// Otherwise, read from writer headers
		value := writer.Header().Get(key)
		L.Push(lua.LString(value))

		return 1
	})
	headersMetatable.RawSetString("__index", headersIndexFunc)

	// __newindex: write headers to writer
	headersNewindexFunc := L.NewFunction(func(L *lua.LState) int {
		key := L.CheckString(2)
		value := L.Get(3)

		if value.Type() == lua.LTString {
			writer.Header().Set(key, value.String())
		}

		return 0
	})
	headersMetatable.RawSetString("__newindex", headersNewindexFunc)

	L.SetMetatable(headersTable, headersMetatable)

	// Add header methods
	addHeaderMethods(L, headersTable, writer)

	return headersTable
}

// addHeaderMethods adds Get and Set methods to the headers table.
func addHeaderMethods(L *lua.LState, headersTable *lua.LTable, writer contracts.ResponseWriter) {
	// Set(key, value) method
	setHeaderMethod := L.NewFunction(func(L *lua.LState) int {
		key := L.CheckString(2)
		value := L.CheckString(3)
		writer.Header().Set(key, value)

		return 0
	})
	headersTable.RawSetString("Set", setHeaderMethod)

	// Get(key) method
	getHeaderMethod := L.NewFunction(func(L *lua.LState) int {
		key := L.CheckString(2)
		value := writer.Header().Get(key)
		L.Push(lua.LString(value))

		return 1
	})
	headersTable.RawSetString("Get", getHeaderMethod)
}

// addResponseMethods adds Write, WriteString, WriteHeader, and Header methods to the response table.
func addResponseMethods(
	L *lua.LState,
	respTable *lua.LTable,
	writer contracts.ResponseWriter,
	headerWritten *bool,
	headersTable *lua.LTable,
) {
	// Header() method - returns headers table
	headerMethod := L.NewFunction(func(L *lua.LState) int {
		L.Push(headersTable)
		return 1
	})
	respTable.RawSetString("Header", headerMethod)

	// Write(data) method
	writeMethod := L.NewFunction(func(L *lua.LState) int {
		data := L.CheckString(2)

		if !*headerWritten {
			writer.WriteHeader(http.StatusOK)
			*headerWritten = true
		}

		n, err := writer.Write([]byte(data))
		if err != nil {
			L.Push(lua.LNumber(0))
			L.Push(lua.LString(err.Error()))
			return 2
		}

		L.Push(lua.LNumber(n))
		L.Push(lua.LNil)

		return 2
	})
	respTable.RawSetString("Write", writeMethod)

	// WriteString(str) method
	writeStringMethod := L.NewFunction(func(L *lua.LState) int {
		str := L.CheckString(2)

		if !*headerWritten {
			writer.WriteHeader(http.StatusOK)
			*headerWritten = true
		}

		n, err := writer.Write([]byte(str))
		if err != nil {
			L.Push(lua.LNumber(0))
			L.Push(lua.LString(err.Error()))
			return 2
		}

		L.Push(lua.LNumber(n))
		L.Push(lua.LNil)

		return 2
	})
	respTable.RawSetString("WriteString", writeStringMethod)

	// WriteHeader(statusCode) method
	writeHeaderMethod := L.NewFunction(func(L *lua.LState) int {
		code := L.CheckInt(2)

		if !*headerWritten {
			writer.WriteHeader(code)
			*headerWritten = true
		}

		return 0
	})
	respTable.RawSetString("WriteHeader", writeHeaderMethod)
}
