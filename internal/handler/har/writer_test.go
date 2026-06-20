package har_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/handler/har"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriter(t *testing.T) {
	t.Run("writes a valid HAR file after Close", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "out.har")

		harWriter := har.NewWriter(path)
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
		harWriter := har.NewWriter(path)

		require.NoError(t, harWriter.Close())
		require.NoError(t, harWriter.Close())
	})

	t.Run("AddEntry does not block when channel is full", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "out.har")
		harWriter := har.NewWriter(path)

		for range 10_000 {
			harWriter.AddEntry(har.Entry{})
		}

		require.NoError(t, harWriter.Close())
	})

	t.Run("file is valid JSON after Close with no entries", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "empty.har")
		harWriter := har.NewWriter(path)

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
		harWriter := har.NewWriter(path)
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

		harWriter := har.NewWriter(path)
		harWriter.AddEntry(har.Entry{})

		require.NoError(t, harWriter.Close())

		_, statErr := os.Stat(path)
		assert.True(t, os.IsNotExist(statErr))
	})

	t.Run("flush handles rename failure gracefully", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "out.har")

		require.NoError(t, os.Mkdir(path, 0o755))

		harWriter := har.NewWriter(path)
		harWriter.AddEntry(har.Entry{})

		require.NoError(t, harWriter.Close())

		fi, statErr := os.Stat(path)
		require.NoError(t, statErr)
		assert.True(t, fi.IsDir())
	})

	t.Run("creates parent directories automatically", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "nested", "deep", "out.har")
		harWriter := har.NewWriter(path)
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
