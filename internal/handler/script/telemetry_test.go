package script_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/log"
)

func TestScriptHandlerLogs(t *testing.T) {
	serve := func(t *testing.T, source string) ([]testutils.LogRecord, error) {
		t.Helper()

		recorded := testutils.Telemetry(t)
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		_, err := serveScript(t, &config.Script{Script: source}, afero.NewMemMapFs(), request)

		return recorded.Records(), err
	}

	t.Run("script executed", func(t *testing.T) {
		records, err := serve(t, `response:WriteHeader(201)`)

		require.NoError(t, err)
		require.Len(t, records, 1)
		assert.Equal(t, "script executed", records[0].Message)
		assert.Equal(t, "inline", records[0].Attrs["script"])
		assert.Equal(t, "201", records[0].Attrs["status"])
	})

	t.Run("script failed", func(t *testing.T) {
		records, err := serve(t, `error("boom")`)

		require.Error(t, err)
		require.Len(t, records, 1)
		assert.Equal(t, log.SeverityWarn, records[0].Severity)
		assert.Equal(t, "lua script failed", records[0].Message)
		assert.Contains(t, records[0].Attrs["error"], "boom")
		assert.Contains(t, records[0].Attrs["stack_trace"], "stack traceback")
	})
}
