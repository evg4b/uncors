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

func (h *Handler) executeScript(writer contracts.ResponseWriter, request *contracts.Request) error {
	luaState := lua.NewState()
	defer luaState.Close()

	h.loadStandardLibraries(luaState)

	reqTable := h.createRequestTable(luaState, request)
	respTable := h.createResponseTable(luaState)

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

	return h.writeResponse(writer, request, luaState)
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

func (h *Handler) createResponseTable(luaState *lua.LState) *lua.LTable {
	respTable := luaState.NewTable()

	respTable.RawSetString("status", lua.LNumber(http.StatusOK))
	respTable.RawSetString("body", lua.LString(""))

	headersTable := luaState.NewTable()
	respTable.RawSetString("headers", headersTable)

	// Add Set(key, value) method to headers table
	setHeaderMethod := luaState.NewFunction(func(L *lua.LState) int {
		key := L.CheckString(2)
		value := L.CheckString(3)
		headersTable.RawSetString(key, lua.LString(value))
		return 0
	})
	headersTable.RawSetString("Set", setHeaderMethod)

	// Add Get(key) method to headers table
	getHeaderMethod := luaState.NewFunction(func(L *lua.LState) int {
		key := L.CheckString(2)
		value := headersTable.RawGetString(key)
		L.Push(value)
		return 1
	})
	headersTable.RawSetString("Get", getHeaderMethod)

	// Add Header() method that returns headers table
	headerMethod := luaState.NewFunction(func(L *lua.LState) int {
		L.Push(headersTable)
		return 1
	})
	respTable.RawSetString("Header", headerMethod)

	// Add Write(data) method
	writeMethod := luaState.NewFunction(func(L *lua.LState) int {
		data := L.CheckString(2)
		bodyValue := respTable.RawGetString("body")
		currentBody := ""
		if bodyValue.Type() == lua.LTString {
			currentBody = bodyValue.String()
		}
		respTable.RawSetString("body", lua.LString(currentBody+data))
		L.Push(lua.LNumber(len(data)))
		L.Push(lua.LNil)
		return 2
	})
	respTable.RawSetString("Write", writeMethod)

	// Add WriteString(str) method
	writeStringMethod := luaState.NewFunction(func(L *lua.LState) int {
		str := L.CheckString(2)
		bodyValue := respTable.RawGetString("body")
		currentBody := ""
		if bodyValue.Type() == lua.LTString {
			currentBody = bodyValue.String()
		}
		respTable.RawSetString("body", lua.LString(currentBody+str))
		L.Push(lua.LNumber(len(str)))
		L.Push(lua.LNil)
		return 2
	})
	respTable.RawSetString("WriteString", writeStringMethod)

	// Add WriteHeader(statusCode) method
	writeHeaderMethod := luaState.NewFunction(func(L *lua.LState) int {
		statusCode := L.CheckInt(2)
		respTable.RawSetString("status", lua.LNumber(statusCode))
		return 0
	})
	respTable.RawSetString("WriteHeader", writeHeaderMethod)

	return respTable
}

func (h *Handler) writeResponse(
	writer contracts.ResponseWriter,
	request *contracts.Request,
	luaState *lua.LState,
) error {
	respTable := luaState.GetGlobal("response")
	if respTable.Type() != lua.LTTable {
		return ErrResponseNotTable
	}

	respTbl, ok := respTable.(*lua.LTable)
	if !ok {
		return ErrInvalidResponseTable
	}

	origin := request.Header.Get("Origin")
	infra.WriteCorsHeaders(writer.Header(), origin)

	headersValue := respTbl.RawGetString("headers")
	if headersValue.Type() == lua.LTTable {
		headersTbl, ok := headersValue.(*lua.LTable)
		if !ok {
			return ErrInvalidHeadersTable
		}
		headersTbl.ForEach(func(key, value lua.LValue) {
			if key.Type() == lua.LTString && value.Type() == lua.LTString {
				writer.Header().Set(key.String(), value.String())
			}
		})
	}

	statusValue := respTbl.RawGetString("status")
	status := http.StatusOK
	if statusValue.Type() == lua.LTNumber {
		status = int(lua.LVAsNumber(statusValue))
	}

	writer.WriteHeader(status)

	bodyValue := respTbl.RawGetString("body")
	if bodyValue.Type() == lua.LTString {
		_, err := writer.Write([]byte(bodyValue.String()))
		if err != nil {
			return fmt.Errorf("failed to write response body: %w", err)
		}
	}

	return nil
}
