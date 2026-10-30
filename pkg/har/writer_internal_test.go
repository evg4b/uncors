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

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWritersSharingAFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "shared.har")
	fs := afero.NewOsFs()

	failures := make([][]error, 2)
	writers := make([]*Writer, 0, len(failures))

	for index := range failures {
		writer := NewWriter(path, WithFileSystem(fs), WithFlushHandler(func(flush Flush) {
			if flush.Err != nil {
				failures[index] = append(failures[index], flush.Err)
			}
		}))
		writers = append(writers, writer)
	}

	var group sync.WaitGroup

	for index, writer := range writers {
		group.Go(func() {
			for range 200 {
				writer.append(Entry{
					Request: Request{Method: http.MethodGet, URL: fmt.Sprintf("http://writer-%d/", index)},
				})
				writer.flush()
			}
		})
	}

	group.Wait()

	for index, writer := range writers {
		require.NoError(t, writer.Close())
		assert.Empty(t, failures[index])
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
	writer := NewWriter("/out.har", WithFileSystem(fs), WithFlushInterval(100*time.Millisecond))

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
	}, time.Second, 20*time.Millisecond)
}

func TestWriterSpoolsFlushedEntries(t *testing.T) {
	fs := afero.NewMemMapFs()
	writer := NewWriter("/out.har", WithFileSystem(fs))

	started := time.Date(2026, time.October, 29, 12, 0, 0, 0, time.UTC)
	entry := func(offset time.Duration, url string) Entry {
		return Entry{StartedDateTime: started.Add(offset), Request: Request{Method: http.MethodGet, URL: url}}
	}

	readURLs := func() []string {
		data, err := afero.ReadFile(fs, "/out.har")
		require.NoError(t, err)

		var archive HAR
		require.NoError(t, json.Unmarshal(data, &archive))

		urls := make([]string, 0, len(archive.Log.Entries))
		for _, entry := range archive.Log.Entries {
			urls = append(urls, entry.Request.URL)
		}

		return urls
	}

	writer.append(entry(0, "/first"))
	writer.append(entry(2*time.Second, "/third"))
	writer.flush()

	assert.Empty(t, writer.encoded)
	assert.Equal(t, []string{"/first", "/third"}, readURLs())

	writer.append(entry(time.Second, "/second"))
	writer.append(entry(3*time.Second, "/fourth"))
	writer.flush()

	assert.Empty(t, writer.encoded)
	assert.Len(t, writer.spooled, 4)
	assert.Equal(t, []string{"/first", "/second", "/third", "/fourth"}, readURLs())

	spool := writer.spool.Name()

	require.NoError(t, writer.Discard())

	exists, err := afero.Exists(fs, spool)
	require.NoError(t, err)
	assert.False(t, exists, "spool file was left behind")
	assert.Equal(t, []string{"/first", "/second", "/third", "/fourth"}, readURLs())
}
