package script

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

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
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

// DefaultTimeout is how long a script may run before it is stopped.
const DefaultTimeout = 30 * time.Second

type Handler struct {
	script  *config.Script
	fs      afero.Fs
	timeout time.Duration
}

func NewHandler(options ...HandlerOption) *Handler {
	return helpers.ApplyOptions(&Handler{timeout: DefaultTimeout}, options)
}

func (h *Handler) ServeHTTP(writer contracts.ResponseWriter, request *contracts.Request) error {
	request, span := telemetry.StartRequest(request, "script.serve")
	defer span.End()

	tracking.Claim(request.Context(), events.HandlerScript)

	var startedAt time.Time

	if telemetry.Enabled() {
		startedAt = time.Now()
	}

	if span.IsRecording() {
		span.SetAttributes(h.attributes()...)
	}

	err := h.executeScript(writer, request)

	if telemetry.Enabled() {
		duration := time.Since(startedAt)
		status := helpers.NormaliseStatusCode(writer.StatusCode())

		if span.IsRecording() {
			span.SetAttributes(semconv.HTTPResponseStatusCode(status))
		}

		telemetry.ScriptExecuted(request.Context(), duration, err)

		if err == nil {
			telemetry.Debug(request.Context(), "script executed",
				h.source(), slog.Int("status", status), slog.Duration("duration", duration))
		}
	}

	if err != nil {
		telemetry.Fail(span, err)

		return err
	}

	return nil
}

// source names the script in logs.
func (h *Handler) source() slog.Attr {
	if h.script.Script != "" {
		return slog.String("script", "inline")
	}

	return slog.String("script", h.script.File)
}

func (h *Handler) attributes() []attribute.KeyValue {
	if h.script.Script != "" {
		return []attribute.KeyValue{
			attribute.String("uncors.script.source", "inline"),
			attribute.Int("uncors.script.bytes", len(h.script.Script)),
		}
	}

	return []attribute.KeyValue{
		attribute.String("uncors.script.source", "file"),
		attribute.String("uncors.script.file", h.script.File),
	}
}

func (h *Handler) executeScript(writer contracts.ResponseWriter, request *contracts.Request) error {
	luaState := newLuaState()
	defer luaState.Close()

	reqTable, err := createRequestTable(luaState, request)
	if err != nil {
		return err
	}

	origin := request.Header.Get("Origin")
	infra.WriteCorsHeaders(writer.Header(), origin)

	respTable := createResponseTable(luaState, writer)

	luaState.SetGlobal("request", reqTable)
	luaState.SetGlobal("response", respTable)

	// The VM checks the context between instructions, so a client that goes
	// away, a server shutdown, or the timeout stops a looping script. A Go
	// function that blocks (os.execute, io.read) is not interrupted.
	ctx, cancel := context.WithTimeoutCause(request.Context(), h.timeout, ErrScriptTimeout)
	defer cancel()

	luaState.SetContext(ctx)

	err = h.runScript(ctx, luaState)
	if err != nil {
		cause := context.Cause(ctx)
		if cause != nil {
			return fmt.Errorf("script error: %w: %w", cause, err)
		}

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
	if err != nil && telemetry.Enabled() {
		h.recordStackTrace(ctx, span, err)
	}

	telemetry.Fail(span, err)

	return err
}

// recordStackTrace keeps the Lua stack trace, which the error returned to the
// server does not carry.
func (h *Handler) recordStackTrace(ctx context.Context, span trace.Span, err error) {
	var apiErr *lua.ApiError
	if !errors.As(err, &apiErr) || apiErr.StackTrace == "" {
		return
	}

	if span.IsRecording() {
		span.SetAttributes(attribute.String("uncors.script.stack_trace", apiErr.StackTrace))
	}

	telemetry.Warn(ctx, "lua script failed",
		h.source(), telemetry.Err(err), slog.String("stack_trace", apiErr.StackTrace))
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

		if span.IsRecording() {
			span.SetAttributes(attribute.Int("uncors.script.bytes", len(source)))
		}
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

// WithTimeout overrides DefaultTimeout.
func WithTimeout(timeout time.Duration) HandlerOption {
	return func(h *Handler) {
		h.timeout = timeout
	}
}
