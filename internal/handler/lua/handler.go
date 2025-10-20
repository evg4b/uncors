package lua

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	lua "github.com/yuin/gopher-lua"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/spf13/afero"
)

type Handler struct {
	script config.LuaScript
	logger contracts.Logger
	fs     afero.Fs
}

var (
	ErrScriptNotDefined     = errors.New("lua script is not defined")
	ErrScriptFileNotFound   = errors.New("lua script file not found")
	ErrBothScriptAndFile    = errors.New("both script and file are defined, only one is allowed")
)

func NewLuaHandler(options ...HandlerOption) *Handler {
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
	// Validate script configuration
	if h.script.Script == "" && h.script.File == "" {
		return ErrScriptNotDefined
	}

	if h.script.Script != "" && h.script.File != "" {
		return ErrBothScriptAndFile
	}

	// Create Lua state
	L := lua.NewState()
	defer L.Close()

	// Load basic Lua modules
	h.loadStandardLibraries(L)

	// Create request and response tables
	reqTable := h.createRequestTable(L, request)
	respTable := h.createResponseTable(L)

	// Set global variables
	L.SetGlobal("request", reqTable)
	L.SetGlobal("response", respTable)

	// Load and execute script
	var err error
	if h.script.Script != "" {
		err = L.DoString(h.script.Script)
	} else {
		scriptContent, readErr := afero.ReadFile(h.fs, h.script.File)
		if readErr != nil {
			return fmt.Errorf("%w: %s", ErrScriptFileNotFound, readErr.Error())
		}
		err = L.DoString(string(scriptContent))
	}

	if err != nil {
		return fmt.Errorf("lua script error: %w", err)
	}

	// Extract response from Lua
	return h.writeResponse(writer, request, L)
}

func (h *Handler) loadStandardLibraries(L *lua.LState) {
	// Load basic Lua libraries
	lua.OpenLibraries(L)
}

func (h *Handler) createRequestTable(L *lua.LState, request *contracts.Request) *lua.LTable {
	reqTable := L.NewTable()

	// Set request properties
	reqTable.RawSetString("method", lua.LString(request.Method))
	reqTable.RawSetString("url", lua.LString(request.URL.String()))
	reqTable.RawSetString("path", lua.LString(request.URL.Path))
	reqTable.RawSetString("query", lua.LString(request.URL.RawQuery))
	reqTable.RawSetString("host", lua.LString(request.Host))
	reqTable.RawSetString("remote_addr", lua.LString(request.RemoteAddr))

	// Set headers
	headersTable := L.NewTable()
	for key, values := range request.Header {
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
	reqTable.RawSetString("headers", headersTable)

	// Set query parameters
	queryTable := L.NewTable()
	for key, values := range request.URL.Query() {
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
	reqTable.RawSetString("query_params", queryTable)

	// Read and set body
	if request.Body != nil {
		body, err := io.ReadAll(request.Body)
		if err == nil {
			reqTable.RawSetString("body", lua.LString(string(body)))
		}
	}

	return reqTable
}

func (h *Handler) createResponseTable(L *lua.LState) *lua.LTable {
	respTable := L.NewTable()

	// Set default values
	respTable.RawSetString("status", lua.LNumber(http.StatusOK))
	respTable.RawSetString("body", lua.LString(""))

	// Create headers table
	headersTable := L.NewTable()
	respTable.RawSetString("headers", headersTable)

	return respTable
}

func (h *Handler) writeResponse(
	writer contracts.ResponseWriter,
	request *contracts.Request,
	L *lua.LState,
) error {
	respTable := L.GetGlobal("response")
	if respTable.Type() != lua.LTTable {
		return errors.New("response must be a table")
	}

	respTbl := respTable.(*lua.LTable)

	// Write CORS headers
	origin := request.Header.Get("Origin")
	infra.WriteCorsHeaders(writer.Header(), origin)

	// Write custom headers from Lua
	headersValue := respTbl.RawGetString("headers")
	if headersValue.Type() == lua.LTTable {
		headersTbl := headersValue.(*lua.LTable)
		headersTbl.ForEach(func(key, value lua.LValue) {
			if key.Type() == lua.LTString && value.Type() == lua.LTString {
				writer.Header().Set(key.String(), value.String())
			}
		})
	}

	// Get status code
	statusValue := respTbl.RawGetString("status")
	status := http.StatusOK
	if statusValue.Type() == lua.LTNumber {
		status = int(lua.LVAsNumber(statusValue))
	}

	// Write status code
	writer.WriteHeader(status)

	// Get and write body
	bodyValue := respTbl.RawGetString("body")
	if bodyValue.Type() == lua.LTString {
		_, err := writer.Write([]byte(bodyValue.String()))
		if err != nil {
			return fmt.Errorf("failed to write response body: %w", err)
		}
	}

	return nil
}
