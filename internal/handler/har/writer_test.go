package har_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/handler/har"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriter(t *testing.T) {
	t.Run("writes a valid HAR file after Close", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "out.har")

		harWriter := har.NewWriter(afero.NewOsFs(), path)
		harWriter.AddEntry(har.Entry{
			StartedDateTime: time.Now(),
			Time:            42,
			Request:         har.Request{Method: "GET", URL: "http://example.com/"},
			Response:        har.Response{Status: 200},
		})

		require.NoError(t, harWriter.Close())

		data, err := os.ReadFile(path)
		require.NoError(t, err)

		var archive har.HAR
		require.NoError(t, json.Unmarshal(data, &archive))

		assert.Equal(t, "1.2", archive.Log.Version)
		assert.Len(t, archive.Log.Entries, 1)
		assert.Equal(t, "GET", archive.Log.Entries[0].Request.Method)
	})

	t.Run("multiple Close calls are safe", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "out.har")
		harWriter := har.NewWriter(afero.NewOsFs(), path)

		require.NoError(t, harWriter.Close())
		require.NoError(t, harWriter.Close())
	})

	t.Run("AddEntry does not block when channel is full", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "out.har")
		harWriter := har.NewWriter(afero.NewOsFs(), path)

		for range 10_000 {
			harWriter.AddEntry(har.Entry{})
		}

		require.NoError(t, harWriter.Close())
	})

	t.Run("file is valid JSON after Close with no entries", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "empty.har")
		harWriter := har.NewWriter(afero.NewOsFs(), path)

		require.NoError(t, harWriter.Close())

		data, err := os.ReadFile(path)
		require.NoError(t, err)

		var archive har.HAR
		require.NoError(t, json.Unmarshal(data, &archive))
		assert.Empty(t, archive.Log.Entries)
	})

	t.Run("flush handles directory creation failure gracefully", func(t *testing.T) {
		dir := t.TempDir()
		blocker := filepath.Join(dir, "blocker")
		require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o600))

		path := filepath.Join(blocker, "sub", "out.har")
		harWriter := har.NewWriter(afero.NewOsFs(), path)
		harWriter.AddEntry(har.Entry{})

		require.NoError(t, harWriter.Close())

		_, statErr := os.Stat(path)
		assert.Error(t, statErr, "HAR file must not be written when parent dir creation fails")
	})

	t.Run("flush handles write failure gracefully", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "out.har")

		require.NoError(t, os.Chmod(dir, 0o500))
		t.Cleanup(func() {
			err := os.Chmod(dir, 0o755)
			require.NoError(t, err)
		})

		harWriter := har.NewWriter(afero.NewOsFs(), path)
		harWriter.AddEntry(har.Entry{})

		require.NoError(t, harWriter.Close())

		_, statErr := os.Stat(path)
		assert.True(t, os.IsNotExist(statErr))
	})

	t.Run("flush handles rename failure gracefully", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "out.har")

		require.NoError(t, os.Mkdir(path, 0o755))

		harWriter := har.NewWriter(afero.NewOsFs(), path)
		harWriter.AddEntry(har.Entry{})

		require.NoError(t, harWriter.Close())

		fi, statErr := os.Stat(path)
		require.NoError(t, statErr)
		assert.True(t, fi.IsDir())
	})

	t.Run("creates parent directories automatically", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "nested", "deep", "out.har")
		harWriter := har.NewWriter(afero.NewOsFs(), path)
		harWriter.AddEntry(har.Entry{
			Request:  har.Request{Method: "GET", URL: "http://example.com/"},
			Response: har.Response{Status: 200},
		})

		require.NoError(t, harWriter.Close())

		data, err := os.ReadFile(path)
		require.NoError(t, err)

		var archive har.HAR
		require.NoError(t, json.Unmarshal(data, &archive))
		assert.Len(t, archive.Log.Entries, 1)
	})
}

// The archive used to be re-serialised in full after every batch, and every
// entry was retained in memory for the process lifetime.
func TestWriterScalesWithSessionLength(t *testing.T) {
	t.Run("records many entries without retaining them", func(t *testing.T) {
		const entries = 2000

		fs := afero.NewMemMapFs()
		path := "/har/out.har"

		harWriter := har.NewWriter(fs, path, har.WithCreatorVersion("1.2.3"))

		for index := range entries {
			harWriter.AddEntry(har.Entry{
				StartedDateTime: time.Now(),
				Request:         har.Request{Method: "GET", URL: fmt.Sprintf("http://example.com/%d", index)},
				Response:        har.Response{Status: 200},
			})
		}

		require.NoError(t, harWriter.Close())

		data, err := afero.ReadFile(fs, path)
		require.NoError(t, err)

		var archive har.HAR
		require.NoError(t, json.Unmarshal(data, &archive))

		assert.Len(t, archive.Log.Entries, entries)
		assert.Equal(t, "1.2.3", archive.Log.Creator.Version, "the archive records the build that produced it")

		// The journal is the writer's own working file and must not be left
		// behind by a clean shutdown.
		_, err = fs.Stat(path + ".jsonl")
		assert.Error(t, err)
	})

	t.Run("bounds a runaway recording", func(t *testing.T) {
		const limit = 10

		fs := afero.NewMemMapFs()
		path := "/har/bounded.har"

		harWriter := har.NewWriter(fs, path, har.WithMaxEntries(limit))

		for range limit * 5 {
			harWriter.AddEntry(har.Entry{Request: har.Request{Method: "GET"}})
		}

		require.NoError(t, harWriter.Close())

		data, err := afero.ReadFile(fs, path)
		require.NoError(t, err)

		var archive har.HAR
		require.NoError(t, json.Unmarshal(data, &archive))

		assert.Len(t, archive.Log.Entries, limit)
	})

	t.Run("two writers over one path do not clobber each other", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		path := "/har/shared.har"

		first := har.NewWriter(fs, path)
		second := har.NewWriter(fs, path)

		assert.NotEqual(t, first.TempPath(), second.TempPath())

		require.NoError(t, first.Close())
		require.NoError(t, second.Close())
	})
}
