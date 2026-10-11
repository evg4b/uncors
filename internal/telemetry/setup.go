package telemetry

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strconv"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

const (
	EnvVar = "UNCORS_TRACE"

	defaultEndpoint = "http://localhost:4318"
	envPrefix       = "OTEL_EXPORTER_OTLP_"
)

// Setup turns tracing on when UNCORS_TRACE is true. The returned function
// flushes and stops the exporters. When tracing is off it builds no SDK
// object, starts no goroutine and sends the default slog and log output to
// discard, so stray records never reach the terminal.
func Setup(ctx context.Context, version string) func(context.Context) error {
	// The OTel defaults print to stderr, which would corrupt the TUI. Export
	// failures are reported by the exporter wrappers below.
	otel.SetLogger(logr.Discard())
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(error) {}))

	noop := func(context.Context) error { return nil }

	if on, _ := strconv.ParseBool(os.Getenv(EnvVar)); !on {
		slog.SetDefault(discard)

		return noop
	}

	shutdown, err := setupExporters(ctx, version)
	if err != nil {
		exportErrors.handle(SignalSpans, err)
		slog.SetDefault(discard)

		return noop
	}

	return shutdown
}

// Endpoint is the OTLP/HTTP traces endpoint the exporter sends to.
func Endpoint() string {
	for _, key := range []string{envPrefix + "TRACES_ENDPOINT", envPrefix + "ENDPOINT"} {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}

	return defaultEndpoint
}

func setupExporters(ctx context.Context, version string) (func(context.Context) error, error) {
	var traceOptions []otlptracehttp.Option

	// The exporters default to HTTPS, which a local collector rarely serves.
	if !configured("TRACES") {
		traceOptions = append(traceOptions, otlptracehttp.WithInsecure())
	}

	traceExporter, err := otlptracehttp.New(ctx, traceOptions...)
	if err != nil {
		return nil, err
	}

	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName("uncors"),
		semconv.ServiceVersion(version),
	)

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(spanExporter{traceExporter}),
		sdktrace.WithResource(res),
	)

	loggerProvider, err := newLoggerProvider(ctx, res)
	if err != nil {
		return nil, errors.Join(err, tracerProvider.Shutdown(ctx))
	}

	otel.SetTracerProvider(tracerProvider)

	if loggerProvider == nil {
		install(tracerProvider, nil)
	} else {
		otel.SetLoggerProvider(loggerProvider)
		install(tracerProvider, loggerProvider)
	}

	// The std log package and third-party slog output reach OTel too.
	slog.SetDefault(Logger())

	return func(ctx context.Context) error {
		err := tracerProvider.Shutdown(ctx)
		if loggerProvider != nil {
			err = errors.Join(err, loggerProvider.Shutdown(ctx))
		}

		return err
	}, nil
}

// newLoggerProvider returns nil when OTEL_LOGS_EXPORTER is "none": Jaeger,
// for one, accepts spans but not logs.
func newLoggerProvider(ctx context.Context, res *resource.Resource) (*sdklog.LoggerProvider, error) {
	if os.Getenv("OTEL_LOGS_EXPORTER") == "none" {
		return nil, nil //nolint:nilnil // no provider is a valid outcome
	}

	var options []otlploghttp.Option

	if !configured("LOGS") {
		options = append(options, otlploghttp.WithInsecure())
	}

	exporter, err := otlploghttp.New(ctx, options...)
	if err != nil {
		return nil, err
	}

	return sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter{exporter})),
		sdklog.WithResource(res),
	), nil
}

// configured reports whether the user chose the signal's endpoint or
// transport security through the OTEL_EXPORTER_OTLP_* variables.
func configured(signal string) bool {
	for _, key := range []string{
		envPrefix + "ENDPOINT",
		envPrefix + signal + "_ENDPOINT",
		envPrefix + "INSECURE",
		envPrefix + signal + "_INSECURE",
	} {
		if os.Getenv(key) != "" {
			return true
		}
	}

	return false
}

type spanExporter struct {
	sdktrace.SpanExporter
}

func (e spanExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	err := e.SpanExporter.ExportSpans(ctx, spans)
	if err != nil {
		exportErrors.handle(SignalSpans, err)
	}

	return err
}

type logExporter struct {
	sdklog.Exporter
}

func (e logExporter) Export(ctx context.Context, records []sdklog.Record) error {
	err := e.Exporter.Export(ctx, records)
	if err != nil {
		exportErrors.handle(SignalLogs, err)
	}

	return err
}
