package telemetry

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// collector is an OTLP/HTTP endpoint that records the paths it receives and
// answers every export with status.
type collector struct {
	mu    sync.Mutex
	paths []string
}

func (c *collector) received() []string {
	c.mu.Lock()
	defer c.mu.Unlock()

	return slices.Compact(slices.Sorted(slices.Values(c.paths)))
}

func startCollector(t *testing.T, status int) *collector {
	t.Helper()

	received := &collector{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		received.mu.Lock()
		received.paths = append(received.paths, request.URL.Path)
		received.mu.Unlock()

		writer.Header().Set("Content-Type", "application/x-protobuf")
		writer.WriteHeader(status)
	}))
	t.Cleanup(server.Close)

	t.Setenv(EnvVar, "true")
	t.Setenv(ConnectionsEnvVar, "")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", server.URL)
	t.Setenv("OTEL_LOGS_EXPORTER", "")
	t.Setenv("OTEL_METRICS_EXPORTER", "")
	t.Setenv("OTEL_METRIC_EXPORT_INTERVAL", "")

	return received
}

// setup runs Setup, records something on every signal, and flushes it. It
// returns the export errors reported and puts back the package state Setup
// changes.
func setup(t *testing.T) ([]reported, error) {
	t.Helper()

	previousLogger := slog.Default()

	var (
		lock sync.Mutex
		got  []reported
	)

	OnExportError(func(signal string, err error) {
		lock.Lock()
		defer lock.Unlock()

		got = append(got, reported{signal, err})
	})

	t.Cleanup(func() {
		current.Store(nil)
		slog.SetDefault(previousLogger)
		OnExportError(nil)
	})

	ctx := context.Background()
	shutdown := Setup(ctx, "v0.0.0")

	require.True(t, Enabled())

	_, span := Start(ctx, "test.span")
	span.End()
	Info(ctx, "test log")
	RequestStarted(ctx)

	err := shutdown(ctx)

	lock.Lock()
	defer lock.Unlock()

	return slices.Clone(got), err
}

func TestSetupExportsEverySignal(t *testing.T) {
	received := startCollector(t, http.StatusOK)

	got, err := setup(t)

	require.NoError(t, err)
	assert.Empty(t, got)
	assert.Equal(t, []string{"/v1/logs", "/v1/metrics", "/v1/traces"}, received.received())
}

func TestSetupSkipsDisabledExporters(t *testing.T) {
	received := startCollector(t, http.StatusOK)
	t.Setenv("OTEL_LOGS_EXPORTER", "none")
	t.Setenv("OTEL_METRICS_EXPORTER", "none")

	_, err := setup(t)

	require.NoError(t, err)
	assert.Equal(t, []string{"/v1/traces"}, received.received())
}

func TestSetupReportsExportErrors(t *testing.T) {
	startCollector(t, http.StatusBadRequest)

	got, err := setup(t)

	require.Error(t, err)

	signals := make([]string, 0, len(got))
	for _, report := range got {
		signals = append(signals, report.signal)
		require.Error(t, report.err)
	}

	assert.ElementsMatch(t, []string{SignalSpans, SignalLogs, SignalMetrics}, signals)
}

func TestSetupReadsTraceConnections(t *testing.T) {
	startCollector(t, http.StatusOK)
	t.Setenv(ConnectionsEnvVar, "true")

	_, err := setup(t)

	require.NoError(t, err)
	assert.True(t, current.Load().providers.TraceConnections)
}
