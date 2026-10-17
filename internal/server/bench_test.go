package server_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/engine"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/handler/proxy"
	"github.com/evg4b/uncors/internal/handler/router"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/internal/telemetry"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/samber/lo"
	"github.com/spf13/afero"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

type stubTransport struct{}

func (stubTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{},
		// A transport's body has no WriteTo, so io.Copy gets no fast path.
		Body:    io.NopCloser(struct{ io.Reader }{strings.NewReader("ok")}),
		Request: request,
	}, nil
}

type discardLogs struct{}

func (discardLogs) Export(context.Context, []sdklog.Record) error { return nil }
func (discardLogs) Shutdown(context.Context) error                { return nil }
func (discardLogs) ForceFlush(context.Context) error              { return nil }

// benchmarkRequest serves a proxied request through the router. The upstream
// client goes through telemetry.WrapTransport, as in infra.MakeHTTPClient.
func benchmarkRequest(b *testing.B) {
	b.Helper()

	mappings := config.Mappings{{From: hosts.Localhost.HTTP(), To: hosts.Github.HTTPS()}}

	generation := lo.Must(engine.New(afero.NewMemMapFs()).BuildGeneration(&config.UncorsConfig{}))

	b.Cleanup(func() { _ = generation.Close() })

	handler := router.NewRouter(mappings,
		router.WithFactory(generation),
		router.WithDefaultHandler(proxy.NewProxyHandler(
			proxy.WithURLReplacerFactory(lo.Must(urlreplacer.NewURLReplacerFactory(mappings))),
			proxy.WithHTTPClient(&http.Client{Transport: telemetry.WrapTransport(stubTransport{})}),
		)),
	)
	instance := server.New(nil, events.NoopPublisher{})

	b.ReportAllocs()

	for b.Loop() {
		request := httptest.NewRequestWithContext(b.Context(), http.MethodGet, "http://localhost/api/users?page=1", nil)
		server.HandleRequest(instance, handler, httptest.NewRecorder(), request)
	}
}

// BenchmarkRequestDisabled measures a request with tracing off: its cost is
// the one the instrumentation must not add to.
func BenchmarkRequestDisabled(b *testing.B) {
	benchmarkRequest(b)
}

// BenchmarkRequestEnabled measures a request with tracing on, through the SDK
// pipelines Setup builds, exporting to nowhere.
func BenchmarkRequestEnabled(b *testing.B) {
	signals := map[string]struct{ logs, metrics bool }{
		"traces":              {},
		"traces+metrics":      {metrics: true},
		"traces+logs":         {logs: true},
		"traces+logs+metrics": {logs: true, metrics: true},
	}

	for name, signal := range signals {
		b.Run(name, func(b *testing.B) {
			tracerProvider := sdktrace.NewTracerProvider(
				sdktrace.WithBatcher(tracetest.NewNoopExporter()),
			)
			providers := telemetry.Providers{Tracer: tracerProvider}

			b.Cleanup(func() { _ = tracerProvider.Shutdown(context.Background()) })

			if signal.logs {
				loggerProvider := sdklog.NewLoggerProvider(
					sdklog.WithProcessor(sdklog.NewBatchProcessor(discardLogs{})),
				)
				providers.Logger = loggerProvider

				b.Cleanup(func() { _ = loggerProvider.Shutdown(context.Background()) })
			}

			if signal.metrics {
				meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(sdkmetric.NewManualReader()))
				providers.Meter = meterProvider

				b.Cleanup(func() { _ = meterProvider.Shutdown(context.Background()) })
			}

			b.Cleanup(telemetry.Use(providers))

			benchmarkRequest(b)
		})
	}
}
