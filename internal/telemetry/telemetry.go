// Package telemetry is the diagnostics layer of the core: OpenTelemetry spans
// and logs, exported over OTLP/HTTP when UNCORS_TRACE is true.
//
// When tracing is off every function returns at once and allocates nothing,
// but variadic and any arguments escape to the heap before the call, so hot
// paths guard attribute building with span.IsRecording() or Enabled().
package telemetry

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync/atomic"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

const scope = "github.com/evg4b/uncors"

type state struct {
	tracer trace.Tracer
	logger *slog.Logger
}

var (
	// nil while tracing is off.
	current atomic.Pointer[state]

	discard             = slog.New(slog.DiscardHandler)
	noopSpan trace.Span = noop.Span{}
)

// Use installs the given providers and turns tracing on. It is the test hook:
// restore puts back what was installed before.
func Use(tracerProvider trace.TracerProvider, loggerProvider log.LoggerProvider) func() {
	previous := current.Load()

	install(tracerProvider, loggerProvider)

	return func() {
		current.Store(previous)
	}
}

// A nil loggerProvider discards logs.
func install(tracerProvider trace.TracerProvider, loggerProvider log.LoggerProvider) {
	logger := discard
	if loggerProvider != nil {
		logger = slog.New(otelslog.NewHandler(scope, otelslog.WithLoggerProvider(loggerProvider)))
	}

	current.Store(&state{
		tracer: tracerProvider.Tracer(scope),
		logger: logger,
	})
}

func Enabled() bool {
	return current.Load() != nil
}

func Start(ctx context.Context, name string) (context.Context, trace.Span) {
	state := current.Load()
	if state == nil {
		return ctx, noopSpan
	}

	return state.tracer.Start(ctx, name) //nolint:spancheck // the caller ends the span
}

func StartKind(ctx context.Context, name string, kind trace.SpanKind) (context.Context, trace.Span) {
	state := current.Load()
	if state == nil {
		return ctx, noopSpan
	}

	return state.tracer.Start(ctx, name, trace.WithSpanKind(kind)) //nolint:spancheck // the caller ends the span
}

// StartServer starts the root span of a new trace for an incoming request:
// the listener's base context may carry the span that started the server.
func StartServer(ctx context.Context, name string) (context.Context, trace.Span) {
	state := current.Load()
	if state == nil {
		return ctx, noopSpan
	}

	//nolint:spancheck // the caller ends the span
	return state.tracer.Start(ctx, name, trace.WithNewRoot(), trace.WithSpanKind(trace.SpanKindServer))
}

// StartRequest starts a span in the request's context and returns the request
// carrying it, so spans started further down nest under it. When tracing is
// off the request is returned unchanged rather than cloned.
func StartRequest(request *http.Request, name string) (*http.Request, trace.Span) {
	state := current.Load()
	if state == nil {
		return request, noopSpan
	}

	ctx, span := state.tracer.Start(request.Context(), name) //nolint:spancheck // the caller ends the span

	return request.WithContext(ctx), span //nolint:spancheck // the caller ends the span
}

// Run runs fn in a span and records the error it returns. It is for cold
// paths: fn is a closure.
func Run(ctx context.Context, name string, fn func(context.Context) error) error {
	ctx, span := Start(ctx, name)
	defer span.End()

	err := fn(ctx)
	Fail(span, err)

	return err
}

// Fail marks the span as failed. A cancelled context is not a failure: the
// client went away.
func Fail(span trace.Span, err error) {
	if err == nil || !span.IsRecording() || errors.Is(err, context.Canceled) {
		return
	}

	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

func Logger() *slog.Logger {
	state := current.Load()
	if state == nil {
		return discard
	}

	return state.logger
}

// The log helpers below are for cold paths: their variadic arguments are
// built by the caller even when tracing is off.

func Debug(ctx context.Context, msg string, attrs ...slog.Attr) {
	logAttrs(ctx, slog.LevelDebug, msg, attrs)
}

func Info(ctx context.Context, msg string, attrs ...slog.Attr) {
	logAttrs(ctx, slog.LevelInfo, msg, attrs)
}

func Warn(ctx context.Context, msg string, attrs ...slog.Attr) {
	logAttrs(ctx, slog.LevelWarn, msg, attrs)
}

func Error(ctx context.Context, msg string, attrs ...slog.Attr) {
	logAttrs(ctx, slog.LevelError, msg, attrs)
}

func logAttrs(ctx context.Context, level slog.Level, msg string, attrs []slog.Attr) {
	state := current.Load()
	if state == nil {
		return
	}

	state.logger.LogAttrs(ctx, level, msg, attrs...)
}
