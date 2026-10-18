package har

import (
	"errors"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/events"
	harlib "github.com/evg4b/uncors/pkg/har"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errFlush = errors.New("disk full")

func TestWriterReportsOneNoticePerFailureStreak(t *testing.T) {
	publisher := &testutils.EventRecorder{}
	reporter := &flushReporter{path: "out.har", publisher: publisher}
	failed := harlib.Flush{Path: "out.har", Start: time.Now(), Err: errFlush}
	written := harlib.Flush{Path: "out.har", Start: time.Now()}

	reporter.flushed(failed)
	reporter.flushed(failed)

	notices := publisher.Notices()
	require.Len(t, notices, 1)
	assert.Equal(t, events.NoticeError, notices[0].Level)
	assert.Equal(t, "HAR: cannot write out.har: disk full", notices[0].Text)

	reporter.flushed(written)
	assert.Len(t, publisher.Notices(), 1)

	reporter.flushed(failed)
	assert.Len(t, publisher.Notices(), 2)
}

func TestNewWriter(t *testing.T) {
	t.Run("publishes a notice when the file cannot be written", func(t *testing.T) {
		publisher := &testutils.EventRecorder{}
		writer := NewWriter("/out.har",
			WithFileSystem(afero.NewReadOnlyFs(afero.NewMemMapFs())),
			WithPublisher(publisher),
		)

		require.NoError(t, writer.Close())

		notices := publisher.Notices()
		require.Len(t, notices, 1)
		assert.Contains(t, notices[0].Text, "HAR: cannot write /out.har")
	})

	t.Run("names uncors and its version as the creator", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		writer := NewWriter("/out.har", WithFileSystem(fs), WithVersion("v1.2.3"))

		require.NoError(t, writer.Close())

		file, err := fs.Open("/out.har")
		require.NoError(t, err)

		defer file.Close()

		archive, err := harlib.Decode(file)
		require.NoError(t, err)
		assert.Equal(t, harlib.Creator{Name: "uncors", Version: "v1.2.3"}, archive.Log.Creator)
	})

	t.Run("requires a file system", func(t *testing.T) {
		assert.PanicsWithValue(t, "HARWriter: File system is not configured", func() {
			NewWriter("out.har")
		})
	})
}
