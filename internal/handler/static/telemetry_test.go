package static_test

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/static"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStaticMiddlewareLogs(t *testing.T) {
	serve := func(t *testing.T, path string) []testutils.LogRecord {
		t.Helper()

		recorded := testutils.Telemetry(t)
		handler := infra.Wrap(static.NewMiddleware(static.WithFileSystem(newStaticFs(t))),
			infra.HandlerFunc(func(contracts.ResponseWriter, *contracts.Request) error { return nil }))

		_, err := serveStatic(t, handler, path)
		require.NoError(t, err)

		return recorded.Records()
	}

	t.Run("served file", func(t *testing.T) {
		records := serve(t, indexJS)

		require.Len(t, records, 1)
		assert.Equal(t, "served static file", records[0].Message)
		assert.Equal(t, map[string]string{
			"file":   indexJS,
			"served": "index.js",
			"bytes":  strconv.Itoa(len(indexJSContent)),
			"status": strconv.Itoa(http.StatusOK),
		}, records[0].Attrs)
	})

	t.Run("missing file", func(t *testing.T) {
		records := serve(t, "/missing.js")

		require.Len(t, records, 1)
		assert.Equal(t, "no static file, passing the request on", records[0].Message)
		assert.Equal(t, "/missing.js", records[0].Attrs["file"])
	})
}
