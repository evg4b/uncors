package script

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/afero"
	lua "github.com/yuin/gopher-lua"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
)

// defaultTimeout bounds how long a script may run. Without one, a `while true do
// end` pins a goroutine and an OS thread until the process is restarted.
const defaultTimeout = 5 * time.Second

type Handler struct {
	script *config.Script
	output contracts.ErrorOutput
	fs     afero.Fs

	proto   *lua.FunctionProto
	timeout time.Duration
}

// NewHandler compiles the script. The source is fixed for the lifetime of a
// configuration, so compiling it here reports a syntax error once, at load, with
// a location — instead of re-reading and re-compiling on every request and
// failing each of them.
func NewHandler(options ...HandlerOption) (*Handler, error) {
	handler := helpers.ApplyOptions(&Handler{}, options)

	source, name, err := handler.source()
	if err != nil {
		return nil, err
	}

	handler.proto, err = compileScript(source, name)
	if err != nil {
		return nil, err
	}

	handler.timeout = handler.script.Timeout
	if handler.timeout <= 0 {
		handler.timeout = defaultTimeout
	}

	return handler, nil
}

func (h *Handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	infra.HandlerFunc(h.Serve).ServeHTTP(writer, request)
}

// Serve is the error returning form of ServeHTTP. ServeHTTP renders a returned
// error as an HTTP error response; callers that want to handle it themselves
// call Serve directly.
func (h *Handler) Serve(writer http.ResponseWriter, request *http.Request) error {
	err := h.executeScript(writer, request)
	if err != nil {
		h.output.Errorf("Script handler error: %v", err)

		return err
	}

	return nil
}

// source returns the script text and a name to report errors against.
func (h *Handler) source() (string, string, error) {
	if h.script == nil {
		return "", "", ErrScriptNotConfigured
	}

	if h.script.Script != "" {
		return h.script.Script, "<inline script>", nil
	}

	if h.script.File == "" {
		return "", "", ErrScriptNotConfigured
	}

	content, err := afero.ReadFile(h.fs, h.script.File)
	if err != nil {
		return "", "", fmt.Errorf("%w: %s", ErrScriptFileNotFound, err.Error())
	}

	return string(content), h.script.File, nil
}

func (h *Handler) executeScript(writer http.ResponseWriter, request *http.Request) error {
	// The script runs under the request's context plus its own deadline, so a
	// client that goes away or a script that never returns both end the run.
	ctx, cancel := context.WithTimeout(request.Context(), h.timeout)
	defer cancel()

	luaState := newLuaState()
	defer luaState.Close()

	luaState.SetContext(ctx)

	origin := request.Header.Get("Origin")
	infra.WriteCorsHeaders(writer.Header(), origin)

	luaState.SetGlobal("request", createRequestTable(luaState, request))
	luaState.SetGlobal("response", createResponseTable(luaState, writer))

	err := h.runScript(luaState)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("%w of %s", ErrScriptTimeout, h.timeout)
		}

		return fmt.Errorf("script error: %w", err)
	}

	return nil
}

func (h *Handler) runScript(luaState *lua.LState) error {
	luaState.Push(luaState.NewFunctionFromProto(h.proto))

	return luaState.PCall(0, lua.MultRet, nil) //nolint:wrapcheck // wrapped by the caller
}

type HandlerOption = func(*Handler)

func WithOutput(output contracts.ErrorOutput) HandlerOption {
	return func(h *Handler) {
		h.output = output
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
