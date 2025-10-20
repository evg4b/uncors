package script

import (
	"fmt"

	"github.com/spf13/afero"
	lua "github.com/yuin/gopher-lua"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/tui"
)

type Handler struct {
	script config.Script
	logger contracts.Logger
	fs     afero.Fs
}

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
	luaState := newLuaState()
	defer luaState.Close()

	origin := request.Header.Get("Origin")
	infra.WriteCorsHeaders(writer.Header(), origin)

	reqTable := createRequestTable(luaState, request)
	respTable := createResponseTable(luaState, writer)

	luaState.SetGlobal("request", reqTable)
	luaState.SetGlobal("response", respTable)

	if err := h.runScript(luaState); err != nil {
		return fmt.Errorf("script error: %w", err)
	}

	return nil
}

func (h *Handler) runScript(luaState *lua.LState) error {
	if h.script.Script != "" {
		return luaState.DoString(h.script.Script)
	}

	scriptContent, err := afero.ReadFile(h.fs, h.script.File)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrScriptFileNotFound, err.Error())
	}

	return luaState.DoString(string(scriptContent))
}
