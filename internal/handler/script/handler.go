package script

import (
	"fmt"

	lua "github.com/yuin/gopher-lua"
	"github.com/spf13/afero"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/tui"
)

// Handler is an HTTP handler that executes Lua scripts to generate responses.
type Handler struct {
	script config.Script
	logger contracts.Logger
	fs     afero.Fs
}

// NewHandler creates a new script handler with the provided options.
func NewHandler(options ...HandlerOption) *Handler {
	return helpers.ApplyOptions(&Handler{}, options)
}

// ServeHTTP handles HTTP requests by executing the configured Lua script.
func (h *Handler) ServeHTTP(writer contracts.ResponseWriter, request *contracts.Request) {
	if err := h.executeScript(writer, request); err != nil {
		infra.HTTPError(writer, err)
		return
	}

	tui.PrintResponse(h.logger, request, writer.StatusCode())
}

// executeScript loads and executes the Lua script, providing request and response objects.
func (h *Handler) executeScript(writer contracts.ResponseWriter, request *contracts.Request) error {
	// Create Lua state
	L := newLuaState()
	defer L.Close()

	// Set CORS headers before script execution
	origin := request.Header.Get("Origin")
	infra.WriteCorsHeaders(writer.Header(), origin)

	// Create request and response tables
	reqTable := createRequestTable(L, request)
	respTable := createResponseTable(L, writer)

	// Set global variables
	L.SetGlobal("request", reqTable)
	L.SetGlobal("response", respTable)

	// Execute script
	if err := h.runScript(L); err != nil {
		return fmt.Errorf("script error: %w", err)
	}

	return nil
}

// runScript executes either inline script or loads from file.
func (h *Handler) runScript(L *lua.LState) error {
	if h.script.Script != "" {
		return L.DoString(h.script.Script)
	}

	scriptContent, err := afero.ReadFile(h.fs, h.script.File)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrScriptFileNotFound, err.Error())
	}

	return L.DoString(string(scriptContent))
}
