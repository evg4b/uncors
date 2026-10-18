package har

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/server"
	harlib "github.com/evg4b/uncors/pkg/har"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/log"
)

func TestWriterLogsFlushes(t *testing.T) {
	recorded := testutils.Telemetry(t)

	// A file where the HAR directory should be makes the flush fail.
	blocker := filepath.Join(t.TempDir(), "blocker")
	require.NoError(t, os.WriteFile(blocker, nil, 0o600))

	path := filepath.Join(blocker, "out.har")
	writer := NewWriter(path, WithFileSystem(afero.NewOsFs()))

	t.Cleanup(func() { _ = writer.Close() })

	// No entry is added, so flush can be driven from the test.
	writer.flush()
	require.NoError(t, os.Remove(blocker))
	writer.flush()

	records := recorded.Records()
	require.Len(t, records, 2)
	assert.Equal(t, log.SeverityError, records[0].Severity)
	assert.Equal(t, "cannot write HAR file", records[0].Message)
	assert.Equal(t, path, records[0].Attrs["file"])
	assert.Equal(t, "HAR file written", records[1].Message)
	assert.Equal(t, "0", records[1].Attrs["entries"])
}

func TestMiddlewareLogsDroppedEntries(t *testing.T) {
	recorded := testutils.Telemetry(t)

	// An unbuffered queue nobody reads is always full.
	writer := &Writer{path: "out.har", publisher: events.NoopPublisher{}, entries: make(chan harlib.Entry)}
	handler := NewMiddleware(WithWriter(writer))

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://localhost/api", nil)
	err := handler.ServeHTTP(server.NewResponseRecorder(httptest.NewRecorder()), request,
		func(contracts.ResponseWriter, *contracts.Request) error { return nil })
	require.NoError(t, err)

	records := recorded.Records()
	require.Len(t, records, 1)
	assert.Equal(t, log.SeverityWarn, records[0].Severity)
	assert.Equal(t, "HAR entry dropped: the write queue is full", records[0].Message)
	assert.Equal(t, "out.har", records[0].Attrs["file"])
}
