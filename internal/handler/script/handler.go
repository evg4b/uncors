package script

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	lua "github.com/yuin/gopher-lua"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/spf13/afero"
)

type Handler struct {
	script config.Script
	logger contracts.Logger
	fs     afero.Fs
}

var (
	ErrScriptFileNotFound   = errors.New("script file not found")
	ErrResponseNotTable     = errors.New("response must be a table")
	ErrInvalidResponseTable = errors.New("invalid response table type")
	ErrInvalidHeadersTable  = errors.New("invalid headers table type")
)

func NewHandler(options ...HandlerOption) *Handler {
	return helpers.ApplyOptions(&Handler{}, options)
}

func (h *Handler) ServeHTTP(writer contracts.ResponseWriter, request *contracts.Request) {
	if err := h.executeScript(writer, request); err != nil {
		infra.HTTPError(writer, err)

		return
	}

	tui.PrintResponse(h.logger, request, writer.StatusCode())
}

type responseState struct {
	statusCode      int
	headers         http.Header
	body            []byte
	headerWritten   bool
	writer          contracts.ResponseWriter
	request         *contracts.Request
}

func (h *Handler) executeScript(writer contracts.ResponseWriter, request *contracts.Request) error {
	luaState := lua.NewState()
	defer luaState.Close()

	h.loadStandardLibraries(luaState)

	respState := &responseState{
		statusCode:    http.StatusOK,
		headers:       make(http.Header),
		body:          []byte{},
		headerWritten: false,
		writer:        writer,
		request:       request,
	}

	reqTable := h.createRequestTable(luaState, request)
	respTable := h.createResponseTable(luaState, respState)

	luaState.SetGlobal("request", reqTable)
	luaState.SetGlobal("response", respTable)

	var err error
	if h.script.Script != "" {
		err = luaState.DoString(h.script.Script)
	} else {
		scriptContent, readErr := afero.ReadFile(h.fs, h.script.File)
		if readErr != nil {
			return fmt.Errorf("%w: %s", ErrScriptFileNotFound, readErr.Error())
		}
		err = luaState.DoString(string(scriptContent))
	}

	if err != nil {
		return fmt.Errorf("script error: %w", err)
	}

	return h.writeResponse(writer, request, respState)
}

func (h *Handler) loadStandardLibraries(luaState *lua.LState) {
	luaState.SetGlobal("_G", luaState.Get(lua.GlobalsIndex))
	luaState.PreloadModule("math", lua.OpenMath)
	luaState.PreloadModule("string", lua.OpenString)
	luaState.PreloadModule("table", lua.OpenTable)
	luaState.PreloadModule("os", lua.OpenOs)
}

func (h *Handler) createRequestTable(luaState *lua.LState, request *contracts.Request) *lua.LTable {
	reqTable := luaState.NewTable()

	reqTable.RawSetString("method", lua.LString(request.Method))
	reqTable.RawSetString("url", lua.LString(request.URL.String()))
	reqTable.RawSetString("path", lua.LString(request.URL.Path))
	reqTable.RawSetString("query", lua.LString(request.URL.RawQuery))
	reqTable.RawSetString("host", lua.LString(request.Host))
	reqTable.RawSetString("remote_addr", lua.LString(request.RemoteAddr))

	headersTable := luaState.NewTable()
	for key, values := range request.Header {
		if len(values) == 1 {
			headersTable.RawSetString(key, lua.LString(values[0]))
		} else {
			valuesList := luaState.NewTable()
			for _, v := range values {
				valuesList.Append(lua.LString(v))
			}
			headersTable.RawSetString(key, valuesList)
		}
	}
	reqTable.RawSetString("headers", headersTable)

	queryTable := luaState.NewTable()
	for key, values := range request.URL.Query() {
		if len(values) == 1 {
			queryTable.RawSetString(key, lua.LString(values[0]))
		} else {
			valuesList := luaState.NewTable()
			for _, v := range values {
				valuesList.Append(lua.LString(v))
			}
			queryTable.RawSetString(key, valuesList)
		}
	}
	reqTable.RawSetString("query_params", queryTable)

	pathVarsTable := luaState.NewTable()
	pathVars := mux.Vars(request)
	for key, value := range pathVars {
		pathVarsTable.RawSetString(key, lua.LString(value))
	}
	reqTable.RawSetString("path_params", pathVarsTable)

	if request.Body != nil {
		body, err := io.ReadAll(request.Body)
		if err == nil {
			reqTable.RawSetString("body", lua.LString(string(body)))
		}
	}

	return reqTable
}

func (h *Handler) createResponseTable(luaState *lua.LState, state *responseState) *lua.LTable {
	respTable := luaState.NewTable()

	// Create metatable to intercept property access
	metatable := luaState.NewTable()

	// __index metamethod for reading properties
	indexFunc := luaState.NewFunction(func(L *lua.LState) int {
		key := L.CheckString(2)

		switch key {
		case "status":
			L.Push(lua.LNumber(state.statusCode))
		case "body":
			L.Push(lua.LString(string(state.body)))
		case "headers":
			L.Push(respTable.RawGetString("headers"))
		default:
			L.Push(respTable.RawGetString(key))
		}

		return 1
	})
	metatable.RawSetString("__index", indexFunc)

	// __newindex metamethod for writing properties
	newindexFunc := luaState.NewFunction(func(L *lua.LState) int {
		key := L.CheckString(2)
		value := L.Get(3)

		switch key {
		case "status":
			if value.Type() == lua.LTNumber {
				state.statusCode = int(lua.LVAsNumber(value))
			}
		case "body":
			if value.Type() == lua.LTString {
				state.body = []byte(value.String())
			}
		default:
			respTable.RawSetString(key, value)
		}

		return 0
	})
	metatable.RawSetString("__newindex", newindexFunc)

	luaState.SetMetatable(respTable, metatable)

	headersTable := luaState.NewTable()

	// Create metatable for headers table to intercept property access
	headersMetatable := luaState.NewTable()

	// __index for reading headers
	headersIndexFunc := luaState.NewFunction(func(L *lua.LState) int {
		key := L.CheckString(2)

		// Check if it's a method first
		method := headersTable.RawGetString(key)
		if method != lua.LNil {
			L.Push(method)
			return 1
		}

		// Otherwise, read from Go headers
		value := state.headers.Get(key)
		L.Push(lua.LString(value))
		return 1
	})
	headersMetatable.RawSetString("__index", headersIndexFunc)

	// __newindex for writing headers
	headersNewindexFunc := luaState.NewFunction(func(L *lua.LState) int {
		key := L.CheckString(2)
		value := L.Get(3)

		if value.Type() == lua.LTString {
			state.headers.Set(key, value.String())
		}

		return 0
	})
	headersMetatable.RawSetString("__newindex", headersNewindexFunc)

	luaState.SetMetatable(headersTable, headersMetatable)

	respTable.RawSetString("headers", headersTable)

	// Add Set(key, value) method to headers table - writes directly to Go headers
	setHeaderMethod := luaState.NewFunction(func(L *lua.LState) int {
		key := L.CheckString(2)
		value := L.CheckString(3)
		state.headers.Set(key, value)
		return 0
	})
	headersTable.RawSetString("Set", setHeaderMethod)

	// Add Get(key) method to headers table - reads from Go headers
	getHeaderMethod := luaState.NewFunction(func(L *lua.LState) int {
		key := L.CheckString(2)
		value := state.headers.Get(key)
		L.Push(lua.LString(value))
		return 1
	})
	headersTable.RawSetString("Get", getHeaderMethod)

	// Add Header() method that returns headers table
	headerMethod := luaState.NewFunction(func(L *lua.LState) int {
		L.Push(headersTable)
		return 1
	})
	respTable.RawSetString("Header", headerMethod)

	// Add Write(data) method - writes directly to Go response state
	writeMethod := luaState.NewFunction(func(L *lua.LState) int {
		data := L.CheckString(2)
		state.body = append(state.body, []byte(data)...)
		L.Push(lua.LNumber(len(data)))
		L.Push(lua.LNil)
		return 2
	})
	respTable.RawSetString("Write", writeMethod)

	// Add WriteString(str) method - writes directly to Go response state
	writeStringMethod := luaState.NewFunction(func(L *lua.LState) int {
		str := L.CheckString(2)
		state.body = append(state.body, []byte(str)...)
		L.Push(lua.LNumber(len(str)))
		L.Push(lua.LNil)
		return 2
	})
	respTable.RawSetString("WriteString", writeStringMethod)

	// Add WriteHeader(statusCode) method - writes directly to Go response state
	writeHeaderMethod := luaState.NewFunction(func(L *lua.LState) int {
		statusCode := L.CheckInt(2)
		state.statusCode = statusCode
		respTable.RawSetString("status", lua.LNumber(statusCode))
		return 0
	})
	respTable.RawSetString("WriteHeader", writeHeaderMethod)

	return respTable
}

func (h *Handler) writeResponse(
	writer contracts.ResponseWriter,
	request *contracts.Request,
	state *responseState,
) error {
	origin := request.Header.Get("Origin")
	infra.WriteCorsHeaders(writer.Header(), origin)

	// Copy headers from state to writer
	for key, values := range state.headers {
		for _, value := range values {
			writer.Header().Add(key, value)
		}
	}

	// Write status code
	writer.WriteHeader(state.statusCode)

	// Write body from state
	if len(state.body) > 0 {
		_, err := writer.Write(state.body)
		if err != nil {
			return fmt.Errorf("failed to write response body: %w", err)
		}
	}

	return nil
}
