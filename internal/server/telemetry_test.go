package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/internal/telemetry"
	"github.com/evg4b/uncors/internal/tracking"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
)

func TestHandleRequestTelemetry(t *testing.T) {
	claiming := infra.HandlerFunc(func(writer contracts.ResponseWriter, request *contracts.Request) error {
		tracking.Claim(request.Context(), events.HandlerMock)

		writer.Header().Set("Access-Control-Allow-Origin", "http://app.local")
		writer.WriteHeader(http.StatusCreated)
		_, err := writer.Write([]byte("body"))

		return err
	})
	handler := infra.CastToHTTPHandler(claiming)

	serve := func(t *testing.T) *httptest.ResponseRecorder {
		t.Helper()

		request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "http://localhost:8080/api?x=1",
			strings.NewReader("payload"))
		request.Header.Set("Origin", "http://app.local")
		request.Header.Set("X-Request-Id", "abc-123")
		request.Header.Set("Authorization", "Bearer secret")

		recorder := httptest.NewRecorder()
		server.HandleRequest(server.New(nil, events.NoopPublisher{}), handler, recorder, request)

		return recorder
	}

	t.Run("tracing off adds no header", func(t *testing.T) {
		assert.Empty(t, serve(t).Header().Get(telemetry.TraceIDHeader))
	})

	t.Run("tracing on", func(t *testing.T) {
		recorded := testutils.Telemetry(t)

		response := serve(t)

		ended := recorded.Spans.Ended()
		require.Len(t, ended, 1)

		root := ended[0]
		assert.Equal(t, "HTTP POST", root.Name())
		assert.Equal(t, root.SpanContext().TraceID().String(), response.Header().Get(telemetry.TraceIDHeader))

		attrs := root.Attributes()
		assert.Contains(t, attrs, attribute.Int64("uncors.request.id", 1))
		assert.Contains(t, attrs, attribute.String("uncors.handler", "MOCK"))
		assert.Contains(t, attrs, attribute.String("server.address", "localhost"))
		assert.Contains(t, attrs, attribute.Int("server.port", 8080))
		assert.Contains(t, attrs, attribute.Int("http.request.body.size", len("payload")))
		assert.Contains(t, attrs, attribute.Int("http.response.status_code", http.StatusCreated))
		assert.Contains(t, attrs, attribute.Int("http.response.body.size", len("body")))
		assert.Contains(t, attrs, attribute.StringSlice("http.request.header.origin", []string{"http://app.local"}))
		assert.Contains(t, attrs, attribute.StringSlice("http.request.header.x-request-id", []string{"abc-123"}))
		assert.Contains(t, attrs,
			attribute.StringSlice("http.response.header.access-control-allow-origin", []string{"http://app.local"}))

		for _, attr := range attrs {
			assert.NotContains(t, attr.Value.String(), "secret", "credentials are never recorded")
		}

		assert.Equal(t, int64(1), recorded.Sum(t, "uncors.requests",
			attribute.String("uncors.handler", "MOCK"),
			attribute.Int("http.response.status_code", http.StatusCreated),
		))
		assert.Equal(t, int64(0), recorded.Sum(t, "http.server.active_requests"))
	})
}
