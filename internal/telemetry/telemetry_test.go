package telemetry_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

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
	spans, logs := testutils.Telemetry(t)

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
	spans, _ := testutils.Telemetry(t)

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
