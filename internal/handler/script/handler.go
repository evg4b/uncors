package script

import (
	"context"
	"fmt"

	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/tracking"

	"github.com/spf13/afero"
	lua "github.com/yuin/gopher-lua"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
)

type Handler struct {
	script *config.Script
	fs     afero.Fs
}

func NewHandler(options ...HandlerOption) *Handler {
	return helpers.ApplyOptions(&Handler{}, options)
}

func (h *Handler) ServeHTTP(writer contracts.ResponseWriter, request *contracts.Request) error {
	request, span := telemetry.StartRequest(request, "script.serve")
	defer span.End()

	tracking.Claim(request.Context(), events.HandlerScript)

	err := h.executeScript(writer, request)
	if err != nil {
		telemetry.Fail(span, err)

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

	err := h.runScript(request.Context(), luaState)
	if err != nil {
		return fmt.Errorf("script error: %w", err)
	}

	return nil
}

func (h *Handler) runScript(ctx context.Context, luaState *lua.LState) error {
	function, err := h.compile(ctx, luaState)
	if err != nil {
		return err
	}

	_, span := telemetry.Start(ctx, "script.lua.execute")
	defer span.End()

	luaState.Push(function)

	err = luaState.PCall(0, lua.MultRet, nil)
	telemetry.Fail(span, err)

	return err
}

func (h *Handler) compile(ctx context.Context, luaState *lua.LState) (*lua.LFunction, error) {
	_, span := telemetry.Start(ctx, "script.lua.compile")
	defer span.End()

	source := h.script.Script

	if source == "" {
		if span.IsRecording() {
			span.SetAttributes(attribute.String("uncors.script.file", h.script.File))
		}

		content, err := afero.ReadFile(h.fs, h.script.File)
		if err != nil {
			err = fmt.Errorf("%w: %s", ErrScriptFileNotFound, err.Error())
			telemetry.Fail(span, err)

			return nil, err
		}

		source = string(content)
	}

	function, err := luaState.LoadString(source)
	telemetry.Fail(span, err)

	return function, err
}

type HandlerOption = func(*Handler)

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
