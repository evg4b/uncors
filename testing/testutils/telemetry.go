package testutils

import (
	"context"
	"slices"
	"testing"

	"github.com/evg4b/uncors/internal/telemetry"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
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
// Options adjust the providers before they are installed.
func Telemetry(t *testing.T, options ...func(*telemetry.Providers)) *Recorded {
	t.Helper()

	recorded := &Recorded{
		Spans:   tracetest.NewSpanRecorder(),
		Logs:    logtest.NewRecorder(),
		metrics: sdkmetric.NewManualReader(),
	}
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorded.Spans))
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(recorded.metrics))

	providers := telemetry.Providers{
		Tracer: tracerProvider,
		Logger: recorded.Logs,
		Meter:  meterProvider,
	}

	for _, option := range options {
		option(&providers)
	}

	restore := telemetry.Use(providers)

	t.Cleanup(func() {
		restore()

		_ = tracerProvider.Shutdown(context.Background())
		_ = meterProvider.Shutdown(context.Background())
	})

	return recorded
}

// TraceConnections makes Telemetry record the connection phases of upstream
// requests.
func TraceConnections(providers *telemetry.Providers) {
	providers.TraceConnections = true
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

// LogRecord is a log the code under test emitted, with its attribute values
// as strings.
type LogRecord struct {
	Severity log.Severity
	Message  string
	Attrs    map[string]string
}

// Records returns the logs emitted so far, in order.
func (r *Recorded) Records() []LogRecord {
	var records []LogRecord

	for _, scopeRecords := range r.Logs.Result() {
		for _, record := range scopeRecords {
			attrs := make(map[string]string, len(record.Attributes))
			for _, attr := range record.Attributes {
				attrs[string(attr.Key)] = attr.Value.String()
			}

			records = append(records, LogRecord{
				Severity: record.Severity,
				Message:  record.Body.AsString(),
				Attrs:    attrs,
			})
		}
	}

	return records
}

// Messages returns the messages of the logs emitted so far, in order.
func (r *Recorded) Messages() []string {
	records := r.Records()
	messages := make([]string, 0, len(records))

	for _, record := range records {
		messages = append(messages, record.Message)
	}

	return messages
}

func containsAll(attrs, match []attribute.KeyValue) bool {
	for _, want := range match {
		if !slices.Contains(attrs, want) {
			return false
		}
	}

	return true
}
