package har

import (
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

	start := time.Date(2026, time.October, 16, 12, 0, 0, 0, time.UTC)
	reporter := &flushReporter{path: "out.har", publisher: events.NoopPublisher{}}
	reporter.flushed(harlib.Flush{Path: "out.har", Start: start, Err: errFlush})
	reporter.flushed(harlib.Flush{Path: "out.har", Start: start, Duration: time.Millisecond})

	records := recorded.Records()
	require.Len(t, records, 2)
	assert.Equal(t, log.SeverityError, records[0].Severity)
	assert.Equal(t, "cannot write HAR file", records[0].Message)
	assert.Equal(t, "out.har", records[0].Attrs["file"])
	assert.Equal(t, "HAR file written", records[1].Message)
	assert.Equal(t, "0", records[1].Attrs["entries"])

	spans := recorded.Spans.Ended()
	require.Len(t, spans, 2)
	assert.Equal(t, "har.flush", spans[1].Name())
	assert.Equal(t, start, spans[1].StartTime())
	assert.Equal(t, start.Add(time.Millisecond), spans[1].EndTime())
}

func TestWriterLogsEntriesItCannotEncode(t *testing.T) {
	recorded := testutils.Telemetry(t)

	writer := NewWriter("/out.har", WithFileSystem(afero.NewMemMapFs()))
	require.True(t, writer.AddEntry(harlib.Entry{Time: math.NaN()}))
	require.NoError(t, writer.Close())

	assert.Contains(t, recorded.Messages(), "cannot encode HAR entry")
}

type fullWriter struct{}

func (fullWriter) AddEntry(harlib.Entry) bool { return false }

func (fullWriter) Path() string { return "out.har" }

func TestMiddlewareLogsDroppedEntries(t *testing.T) {
	recorded := testutils.Telemetry(t)

	handler := NewMiddleware(WithWriter(fullWriter{}))

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
