package infra_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/telemetry"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestHTTPClientTracing(t *testing.T) {
	serve := func(t *testing.T, options ...func(*telemetry.Providers)) sdktrace.ReadOnlySpan {
		t.Helper()

		recorded := testutils.Telemetry(t, options...)

		var received http.Header

		upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			received = request.Header.Clone()

			writer.WriteHeader(http.StatusNoContent)
		}))
		defer upstream.Close()

		ctx, parent := telemetry.Start(context.Background(), "test.parent")

		request, err := http.NewRequestWithContext(ctx, http.MethodGet, upstream.URL, nil)
		require.NoError(t, err)

		httpClient, err := infra.MakeHTTPClient("")
		require.NoError(t, err)

		response, err := httpClient.Do(request)
		require.NoError(t, err)
		require.NoError(t, response.Body.Close())
		parent.End()

		assert.Empty(t, received.Get("Traceparent"), "trace context must not reach the upstream")
		assert.Empty(t, received.Get("Tracestate"))
		assert.Empty(t, received.Get("Baggage"))

		ended := recorded.Spans.Ended()
		client := slices.IndexFunc(ended, func(span sdktrace.ReadOnlySpan) bool {
			return span.Name() == "HTTP GET"
		})
		require.GreaterOrEqual(t, client, 0, "the request has a client span")
		assert.Equal(t, parent.SpanContext().SpanID(), ended[client].Parent().SpanID())

		return ended[client]
	}

	connected := func(span sdktrace.ReadOnlySpan) bool {
		return slices.ContainsFunc(span.Events(), func(event sdktrace.Event) bool {
			return event.Name == "http.getconn.done"
		})
	}

	t.Run("connections are not traced by default", func(t *testing.T) {
		assert.False(t, connected(serve(t)))
	})

	t.Run("connections are traced on request", func(t *testing.T) {
		assert.True(t, connected(serve(t, testutils.TraceConnections)))
	})
}
