package telemetry

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

const (
	EnvVar            = "UNCORS_TRACE"
	ConnectionsEnvVar = "UNCORS_TRACE_CONNECTIONS"

	defaultEndpoint = "http://localhost:4318"
	envPrefix       = "OTEL_EXPORTER_OTLP_"
	metricInterval  = 10 * time.Second
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
	shutdowns := []func(context.Context) error{tracerProvider.Shutdown}
	shutdown := func(ctx context.Context) error {
		errs := make([]error, 0, len(shutdowns))
		for _, stop := range shutdowns {
			errs = append(errs, stop(ctx))
		}

		return errors.Join(errs...)
	}

	traceConnections, _ := strconv.ParseBool(os.Getenv(ConnectionsEnvVar))
	providers := Providers{Tracer: tracerProvider, TraceConnections: traceConnections}
	otel.SetTracerProvider(tracerProvider)

	loggerProvider, err := newLoggerProvider(ctx, res)
	if err != nil {
		return nil, errors.Join(err, shutdown(ctx))
	}

	if loggerProvider != nil {
		providers.Logger = loggerProvider
		shutdowns = append(shutdowns, loggerProvider.Shutdown)
		otel.SetLoggerProvider(loggerProvider)
	}

	meterProvider, err := newMeterProvider(ctx, res)
	if err != nil {
		return nil, errors.Join(err, shutdown(ctx))
	}

	if meterProvider != nil {
		providers.Meter = meterProvider
		shutdowns = append(shutdowns, meterProvider.Shutdown)
		otel.SetMeterProvider(meterProvider)

		err = runtime.Start(runtime.WithMeterProvider(meterProvider))
		if err != nil {
			return nil, errors.Join(err, shutdown(ctx))
		}
	}

	install(providers)
	// The std log package and third-party slog output reach OTel too.
	slog.SetDefault(Logger())

	return shutdown, nil
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

// newMeterProvider returns nil when OTEL_METRICS_EXPORTER is "none".
func newMeterProvider(ctx context.Context, res *resource.Resource) (*sdkmetric.MeterProvider, error) {
	if os.Getenv("OTEL_METRICS_EXPORTER") == "none" {
		return nil, nil //nolint:nilnil // no provider is a valid outcome
	}

	var options []otlpmetrichttp.Option

	if !configured("METRICS") {
		options = append(options, otlpmetrichttp.WithInsecure())
	}

	exporter, err := otlpmetrichttp.New(ctx, options...)
	if err != nil {
		return nil, err
	}

	var readerOptions []sdkmetric.PeriodicReaderOption

	// The SDK default of a minute is long for watching a dev proxy.
	if os.Getenv("OTEL_METRIC_EXPORT_INTERVAL") == "" {
		readerOptions = append(readerOptions, sdkmetric.WithInterval(metricInterval))
	}

	return sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter{exporter}, readerOptions...)),
		sdkmetric.WithResource(res),
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

type metricExporter struct {
	sdkmetric.Exporter
}

func (e metricExporter) Export(ctx context.Context, metrics *metricdata.ResourceMetrics) error {
	err := e.Exporter.Export(ctx, metrics)
	if err != nil {
		exportErrors.handle(SignalMetrics, err)
	}

	return err
}
