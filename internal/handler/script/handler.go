package script

import (
	"fmt"
	"log/slog"

	"github.com/spf13/afero"
	lua "github.com/yuin/gopher-lua"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/pkg/urlt"
)

type Handler struct {
	script *config.Script
	logger *slog.Logger
	fs     afero.Fs
}

func NewHandler(options ...HandlerOption) *Handler {
	return helpers.ApplyOptions(&Handler{logger: slog.Default()}, options)
}

func (h *Handler) ServeHTTP(writer contracts.ResponseWriter, request *contracts.Request) error {
	err := h.executeScript(writer, request)
	if err != nil {
		h.logger.Error("script handler error", "url", urlt.URL_String(request.URL), "error", err)

		return err
	}

	return nil
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

	err := h.runScript(luaState)
	if err != nil {
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

type HandlerOption = func(*Handler)

func WithLogger(logger *slog.Logger) HandlerOption {
	return func(h *Handler) {
		h.logger = logger
	}
}

func WithScript(script *config.Script) HandlerOption {
	return func(h *Handler) {
		h.script = script
	}
}

func WithFileSystem(fs afero.Fs) HandlerOption {
	return func(h *Handler) {
		h.fs = fs
	}
}
