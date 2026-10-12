package telemetry_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/telemetry"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/trace"
)

var errTest = errors.New("test error")

func TestDisabledDoesNotAllocate(t *testing.T) {
	require.False(t, telemetry.Enabled())

	ctx := context.Background()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://localhost/", nil)

	cases := map[string]func(){
		"Start and End": func() {
			_, span := telemetry.Start(ctx, "test.span")
			span.End()
		},
		"StartKind": func() {
			_, span := telemetry.StartKind(ctx, "test.span", trace.SpanKindClient)
			span.End()
		},
		"StartServer": func() {
			_, span := telemetry.StartServer(ctx, "test.span")
			span.End()
		},
		"StartRequest": func() {
			_, span := telemetry.StartRequest(request, "test.span")
			span.End()
		},
		"Fail": func() {
			_, span := telemetry.Start(ctx, "test.span")
			telemetry.Fail(span, errTest)
		},
		"guarded SetAttributes": func() {
			_, span := telemetry.Start(ctx, "test.span")
			if span.IsRecording() {
				span.SetAttributes(attribute.String("key", request.URL.String()))
			}
		},
		"metrics": func() {
			telemetry.RequestStarted(ctx)
			telemetry.CacheLookup(ctx, true)
			telemetry.CacheStored(ctx, 10)
			telemetry.MockServed(ctx, http.StatusOK, time.Second, false)
			telemetry.ScriptExecuted(ctx, time.Second, errTest)
			telemetry.StaticRequest(ctx, telemetry.ResultServed)
			telemetry.Rewritten(ctx)
			telemetry.HAREntry(ctx, true)
			telemetry.HARFlushed(ctx, nil)
			telemetry.ConfigReloaded(ctx, errTest)
			telemetry.TLSCertificate(ctx, nil)
		},
		"guarded Debug": func() {
			if telemetry.Enabled() {
				telemetry.Debug(ctx, "message", slog.String("key", request.URL.String()))
			}
		},
	}

	for name, fn := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Zero(t, testing.AllocsPerRun(100, fn))
		})
	}
}

func TestStartRequestKeepsTheRequestWhenDisabled(t *testing.T) {
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://localhost/", nil)

	got, span := telemetry.StartRequest(request, "test.span")

	assert.Same(t, request, got)
	assert.False(t, span.IsRecording())
}

func TestEnabledRecordsSpansAndLogs(t *testing.T) {
	recorded := testutils.Telemetry(t)
	spans, logs := recorded.Spans, recorded.Logs

	require.True(t, telemetry.Enabled())

	ctx, parent := telemetry.Start(context.Background(), "test.parent")
	request := httptest.NewRequestWithContext(ctx, http.MethodGet, "http://localhost/", nil)

	request, child := telemetry.StartRequest(request, "test.child")
	telemetry.Debug(request.Context(), "inside child")
	telemetry.Fail(child, errTest)
	child.End()
	parent.End()

	ended := spans.Ended()
	require.Len(t, ended, 2)
	assert.Equal(t, "test.child", ended[0].Name())
	assert.Equal(t, parent.SpanContext().SpanID(), ended[0].Parent().SpanID())
	assert.Equal(t, codes.Error, ended[0].Status().Code)
	assert.Equal(t, "test.parent", ended[1].Name())

	var records []string

	for _, scopeRecords := range logs.Result() {
		for _, record := range scopeRecords {
			records = append(records, record.Body.AsString())
			assert.Equal(t, log.SeverityDebug, record.Severity)
			assert.Equal(t, child.SpanContext().TraceID(), trace.SpanContextFromContext(record.Context).TraceID())
		}
	}

	assert.Equal(t, []string{"inside child"}, records)
}

func TestFailIgnoresCancellation(t *testing.T) {
	spans := testutils.Telemetry(t).Spans

	_, span := telemetry.Start(context.Background(), "test.span")
	telemetry.Fail(span, context.Canceled)
	telemetry.Fail(span, nil)
	span.End()

	require.Len(t, spans.Ended(), 1)
	assert.Equal(t, codes.Unset, spans.Ended()[0].Status().Code)
}

func TestSetupDisabled(t *testing.T) {
	t.Setenv(telemetry.EnvVar, "")

	shutdown := telemetry.Setup(context.Background(), "v0.0.0")

	assert.False(t, telemetry.Enabled())
	assert.False(t, telemetry.Logger().Enabled(context.Background(), slog.LevelError))
	assert.NoError(t, shutdown(context.Background()))
}

func TestEndpoint(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	assert.Equal(t, "http://localhost:4318", telemetry.Endpoint())

	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://collector:4318")
	assert.Equal(t, "http://collector:4318", telemetry.Endpoint())

	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "http://traces:4318/v1/traces")
	assert.Equal(t, "http://traces:4318/v1/traces", telemetry.Endpoint())
}

func TestWrapTransportWhenDisabled(t *testing.T) {
	transport := &http.Transport{}

	assert.Same(t, transport, telemetry.WrapTransport(transport))
}

func TestMetrics(t *testing.T) {
	recorded := testutils.Telemetry(t)
	ctx := context.Background()

	telemetry.CacheLookup(ctx, true)
	telemetry.CacheLookup(ctx, false)
	telemetry.CacheLookup(ctx, false)
	telemetry.CacheStored(ctx, 128)
	telemetry.MockServed(ctx, http.StatusOK, time.Second, false)
	telemetry.MockServed(ctx, http.StatusServiceUnavailable, time.Second, true)
	telemetry.StaticRequest(ctx, telemetry.ResultFallThrough)
	telemetry.HAREntry(ctx, false)
	telemetry.ConfigReloaded(ctx, errTest)
	telemetry.RequestStarted(ctx)
	telemetry.RequestStarted(ctx)
	telemetry.RequestFinished(ctx, telemetry.RequestMetrics{
		Method: http.MethodGet, Handler: "MOCK", Status: http.StatusOK, Duration: time.Millisecond, ResponseBytes: 4,
	})

	result := attribute.Key("uncors.result")

	assert.Equal(t, int64(1), recorded.Sum(t, "uncors.cache.lookups", result.String(telemetry.ResultHit)))
	assert.Equal(t, int64(2), recorded.Sum(t, "uncors.cache.lookups", result.String(telemetry.ResultMiss)))
	assert.Equal(t, int64(1), recorded.Sum(t, "uncors.cache.stores"))
	assert.Equal(t, int64(128), recorded.Sum(t, "uncors.cache.stored_bytes"))
	assert.Equal(t, int64(1), recorded.Sum(t, "uncors.mock.responses", result.String(telemetry.ResultCancelled)))
	assert.Equal(t, int64(1), recorded.Sum(t, "uncors.static.requests", result.String(telemetry.ResultFallThrough)))
	assert.Equal(t, int64(1), recorded.Sum(t, "uncors.har.entries", result.String(telemetry.ResultDropped)))
	assert.Equal(t, int64(1), recorded.Sum(t, "uncors.config.reloads", result.String(telemetry.ResultError)))
	assert.Equal(t, int64(1), recorded.Sum(t, "uncors.requests", attribute.String("uncors.handler", "MOCK")))
	assert.Equal(t, int64(1), recorded.Sum(t, "http.server.active_requests"))
}

func TestProtocolVersion(t *testing.T) {
	cases := map[string][2]int{
		"1.0": {1, 0},
		"1.1": {1, 1},
		"2":   {2, 0},
		"3":   {3, 0},
	}

	for expected, version := range cases {
		t.Run(expected, func(t *testing.T) {
			assert.Equal(t, expected, telemetry.ProtocolVersion(version[0], version[1]))
		})
	}
}
