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
	"strings"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

const scope = "github.com/evg4b/uncors"

type state struct {
	providers Providers
	tracer    trace.Tracer
	logger    *slog.Logger
	metrics   *instruments
}

// Providers are the OTel providers telemetry records to. A nil Logger
// discards logs and a nil Meter discards metrics.
type Providers struct {
	Tracer trace.TracerProvider
	Logger log.LoggerProvider
	Meter  metric.MeterProvider

	// TraceConnections records the DNS, connection, and TLS phases of
	// upstream requests, at a cost to every one of them.
	TraceConnections bool
}

var (
	// nil while tracing is off.
	current atomic.Pointer[state]

	discard             = slog.New(slog.DiscardHandler)
	noopSpan trace.Span = noop.Span{}
)

// Use installs the given providers and turns tracing on. It is the test hook:
// restore puts back what was installed before.
func Use(providers Providers) func() {
	previous := current.Load()

	install(providers)

	return func() {
		current.Store(previous)
	}
}

func install(providers Providers) {
	if providers.Meter == nil {
		providers.Meter = metricnoop.NewMeterProvider()
	}

	logger := discard
	if providers.Logger != nil {
		logger = slog.New(otelslog.NewHandler(scope, otelslog.WithLoggerProvider(providers.Logger)))
	}

	current.Store(&state{
		providers: providers,
		tracer:    providers.Tracer.Tracer(scope),
		logger:    logger,
		metrics:   newInstruments(providers.Meter.Meter(scope)),
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

	return state.tracer.Start(ctx, name)
}

func StartAt(ctx context.Context, name string, start time.Time) (context.Context, trace.Span) {
	state := current.Load()
	if state == nil {
		return ctx, noopSpan
	}

	return state.tracer.Start(ctx, name, trace.WithTimestamp(start))
}

func StartKind(ctx context.Context, name string, kind trace.SpanKind) (context.Context, trace.Span) {
	state := current.Load()
	if state == nil {
		return ctx, noopSpan
	}

	return state.tracer.Start(ctx, name, trace.WithSpanKind(kind))
}

// StartServer starts the root span of a new trace for an incoming request:
// the listener's base context may carry the span that started the server.
func StartServer(ctx context.Context, name string) (context.Context, trace.Span) {
	state := current.Load()
	if state == nil {
		return ctx, noopSpan
	}

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

	ctx, span := state.tracer.Start(request.Context(), name)

	return request.WithContext(ctx), span
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

func Err(err error) slog.Attr {
	return slog.String("error", err.Error())
}

// inlineAttrs is how many attributes logAttrs escapes without allocating.
const inlineAttrs = 8

func logAttrs(ctx context.Context, level slog.Level, msg string, attrs []slog.Attr) {
	state := current.Load()
	if state == nil {
		return
	}

	// String values often come from the request: URLs, headers, and errors
	// that quote them. Their line breaks are escaped so they cannot forge
	// records in a backend that stores logs as lines.
	var buffer [inlineAttrs]slog.Attr

	escaped := buffer[:0]

	for _, attr := range attrs {
		escaped = append(escaped, escapeAttr(attr))
	}

	state.logger.LogAttrs(ctx, level, msg, escaped...)
}

// escapeAttr rebuilds attr from its parts, so no value reaches the logger
// without going through escapeLineBreaks or a type that cannot hold a line
// break. Groups and other values are logged as their escaped text.
func escapeAttr(attr slog.Attr) slog.Attr {
	value := attr.Value

	switch value.Kind() {
	case slog.KindInt64:
		return slog.Int64(attr.Key, value.Int64())
	case slog.KindUint64:
		return slog.Uint64(attr.Key, value.Uint64())
	case slog.KindFloat64:
		return slog.Float64(attr.Key, value.Float64())
	case slog.KindBool:
		return slog.Bool(attr.Key, value.Bool())
	case slog.KindDuration:
		return slog.Duration(attr.Key, value.Duration())
	case slog.KindTime:
		return slog.Time(attr.Key, value.Time())
	case slog.KindString, slog.KindAny, slog.KindGroup, slog.KindLogValuer:
		// Logged as escaped text below.
	}

	return slog.String(escapeLineBreaks(attr.Key), escapeLineBreaks(value.String()))
}

func escapeLineBreaks(value string) string {
	value = strings.ReplaceAll(value, "\n", `\n`)

	return strings.ReplaceAll(value, "\r", `\r`)
}
