package testutils

import (
	"context"
	"testing"

	"github.com/evg4b/uncors/internal/telemetry"
	"go.opentelemetry.io/otel/log/logtest"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// Telemetry turns tracing on with in-memory providers for the rest of the
// test. It changes package state, so the test must not run in parallel.
func Telemetry(t *testing.T) (*tracetest.SpanRecorder, *logtest.Recorder) {
	t.Helper()

	spans := tracetest.NewSpanRecorder()
	logs := logtest.NewRecorder()
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spans))

	restore := telemetry.Use(tracerProvider, logs)

	t.Cleanup(func() {
		restore()

		_ = tracerProvider.Shutdown(context.Background())
	})

	return spans, logs
}
