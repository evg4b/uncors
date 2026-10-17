package har

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriterReportsOneNoticePerFailureStreak(t *testing.T) {
	// A file where the HAR directory should be makes every flush fail.
	blocker := filepath.Join(t.TempDir(), "blocker")
	block := func() { require.NoError(t, os.WriteFile(blocker, nil, 0o600)) }

	block()

	publisher := &testutils.EventRecorder{}
	path := filepath.Join(blocker, "out.har")
	writer := NewWriter(path, WithFileSystem(afero.NewOsFs()), WithPublisher(publisher))

	// No entry is added, so the writer goroutine stays idle and flush can be
	// driven from the test.
	writer.flush()
	writer.flush()

	notices := publisher.Notices()
	require.Len(t, notices, 1)
	assert.Equal(t, events.NoticeError, notices[0].Level)
	assert.Contains(t, notices[0].Text, "HAR: cannot write "+path)

	require.NoError(t, os.Remove(blocker))
	writer.flush()
	assert.FileExists(t, path)
	assert.Len(t, publisher.Notices(), 1)

	require.NoError(t, os.RemoveAll(blocker))
	block()
	writer.flush()
	assert.Len(t, publisher.Notices(), 2)

	require.NoError(t, writer.Close())
	assert.Len(t, publisher.Notices(), 2, "the failing flush on Close continues the streak")
}

func TestWritersSharingAFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "shared.har")
	fs := afero.NewOsFs()

	publishers := []*testutils.EventRecorder{{}, {}}
	writers := make([]*Writer, 0, len(publishers))

	for _, publisher := range publishers {
		writer := NewWriter(path, WithFileSystem(fs), WithPublisher(publisher))
		writers = append(writers, writer)
	}

	var group sync.WaitGroup

	for index, writer := range writers {
		group.Go(func() {
			for range 200 {
				writer.append(Entry{Request: Request{Method: http.MethodGet, URL: fmt.Sprintf("http://writer-%d/", index)}})
				writer.flush()
			}
		})
	}

	group.Wait()

	for index, writer := range writers {
		require.NoError(t, writer.Close())
		assert.Empty(t, publishers[index].Notices())
	}

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var archive HAR
	require.NoError(t, json.Unmarshal(data, &archive))
	assert.Len(t, archive.Log.Entries, 200)

	files, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, "shared.har", files[0].Name())
}

func TestWriterFlushesOnInterval(t *testing.T) {
	fs := afero.NewMemMapFs()
	writer := NewWriter("/out.har", WithFileSystem(fs))

	t.Cleanup(func() { _ = writer.Close() })

	assert.True(t, writer.AddEntry(Entry{Request: Request{Method: http.MethodPost}}))

	assert.EventuallyWithT(t, func(collect *assert.CollectT) {
		data, err := afero.ReadFile(fs, "/out.har")
		if !assert.NoError(collect, err) {
			return
		}

		var archive HAR
		if assert.NoError(collect, json.Unmarshal(data, &archive)) {
			assert.Len(collect, archive.Log.Entries, 1)
		}
	}, 3*flushInterval, 50*time.Millisecond)
}
