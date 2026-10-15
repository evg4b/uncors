package infra_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/log"
)

var errHandler = errors.New("handler failed")

func TestCastToHTTPHandlerLogsFailures(t *testing.T) {
	failing := infra.HandlerFunc(func(contracts.ResponseWriter, *contracts.Request) error {
		return errHandler
	})

	serve := func(t *testing.T, ctx context.Context) []testutils.LogRecord {
		t.Helper()

		recorded := testutils.Telemetry(t)
		request := httptest.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:8080/api", nil)
		infra.CastToHTTPHandler(failing).ServeHTTP(server.NewResponseRecorder(httptest.NewRecorder()), request)

		return recorded.Records()
	}

	t.Run("handler error", func(t *testing.T) {
		records := serve(t, t.Context())

		require.Len(t, records, 1)
		assert.Equal(t, log.SeverityError, records[0].Severity)
		assert.Equal(t, "request failed", records[0].Message)
		assert.Equal(t, map[string]string{
			"method": http.MethodGet,
			"url":    "http://localhost:8080/api",
			"error":  errHandler.Error(),
		}, records[0].Attrs)
	})

	t.Run("cancelled request", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		records := serve(t, ctx)

		require.Len(t, records, 1)
		assert.Equal(t, log.SeverityDebug, records[0].Severity)
		assert.Equal(t, "request cancelled by the client", records[0].Message)
	})
}
