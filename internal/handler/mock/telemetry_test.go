package mock_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/handler/mock"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func eventNames(span sdktrace.ReadOnlySpan) []string {
	names := make([]string, 0, len(span.Events()))
	for _, event := range span.Events() {
		names = append(names, event.Name)
	}

	return names
}

func TestMockTelemetry(t *testing.T) {
	response := &config.Response{Code: http.StatusAccepted, Raw: textContent, Delay: time.Second}

	serve := func(t *testing.T, ctx context.Context, after <-chan time.Time) sdktrace.ReadOnlySpan {
		t.Helper()

		recorded := testutils.Telemetry(t)
		handler := mock.NewMockHandler(
			mock.WithResponse(response),
			mock.WithAfter(func(time.Duration) <-chan time.Time { return after }),
		)

		request := httptest.NewRequestWithContext(ctx, http.MethodGet, "http://localhost/mock", nil)
		require.NoError(t, handler.ServeHTTP(server.NewResponseRecorder(httptest.NewRecorder()), request))

		ended := recorded.Spans.Ended()
		require.Len(t, ended, 1)
		assert.Equal(t, "mock.serve", ended[0].Name())

		return ended[0]
	}

	t.Run("delay finished", func(t *testing.T) {
		after := make(chan time.Time, 1)
		after <- time.Now()

		span := serve(t, t.Context(), after)

		assert.Equal(t, []string{"uncors.mock.delay_started", "uncors.mock.delay_finished"}, eventNames(span))
		assert.Contains(t, span.Attributes(), attribute.Int("uncors.mock.status", http.StatusAccepted))
		assert.Contains(t, span.Attributes(), attribute.String("uncors.mock.delay", "1s"))
		assert.Contains(t, span.Attributes(), attribute.String("uncors.mock.source", "raw"))
		assert.Contains(t, span.Attributes(), attribute.Int("uncors.mock.raw_bytes", len(textContent)))
	})

	t.Run("delay cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		span := serve(t, ctx, make(chan time.Time))

		assert.Equal(t, []string{"uncors.mock.delay_started", "uncors.mock.delay_cancelled"}, eventNames(span))
	})
}
