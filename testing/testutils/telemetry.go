package testutils

import (
	"context"
	"slices"
	"testing"

	"github.com/evg4b/uncors/internal/telemetry"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log/logtest"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// Recorded holds what the code under test recorded while tracing was on.
type Recorded struct {
	Spans   *tracetest.SpanRecorder
	Logs    *logtest.Recorder
	metrics *sdkmetric.ManualReader
}

// Telemetry turns tracing on with in-memory providers for the rest of the
// test. It changes package state, so the test must not run in parallel.
func Telemetry(t *testing.T) *Recorded {
	t.Helper()

	recorded := &Recorded{
		Spans:   tracetest.NewSpanRecorder(),
		Logs:    logtest.NewRecorder(),
		metrics: sdkmetric.NewManualReader(),
	}
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorded.Spans))
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(recorded.metrics))

	restore := telemetry.Use(telemetry.Providers{
		Tracer: tracerProvider,
		Logger: recorded.Logs,
		Meter:  meterProvider,
	})

	t.Cleanup(func() {
		restore()

		_ = tracerProvider.Shutdown(context.Background())
		_ = meterProvider.Shutdown(context.Background())
	})

	return recorded
}

// Sum returns the total of the counter or up-down counter called name over
// the points whose attributes include every one of match.
func (r *Recorded) Sum(t *testing.T, name string, match ...attribute.KeyValue) int64 {
	t.Helper()

	var data metricdata.ResourceMetrics
	require.NoError(t, r.metrics.Collect(context.Background(), &data))

	var total int64

	for _, scope := range data.ScopeMetrics {
		for _, instrument := range scope.Metrics {
			sum, ok := instrument.Data.(metricdata.Sum[int64])
			if instrument.Name != name || !ok {
				continue
			}

			for _, point := range sum.DataPoints {
				if containsAll(point.Attributes.ToSlice(), match) {
					total += point.Value
				}
			}
		}
	}

	return total
}

func containsAll(attrs, match []attribute.KeyValue) bool {
	for _, want := range match {
		if !slices.Contains(attrs, want) {
			return false
		}
	}

	return true
}
