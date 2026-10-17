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

		harWriter := har.NewWriter(path, har.WithFileSystem(afero.NewOsFs()))
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
		harWriter := har.NewWriter(path, har.WithFileSystem(afero.NewOsFs()))

		require.NoError(t, harWriter.Close())
		require.NoError(t, harWriter.Close())
	})

	t.Run("AddEntry does not block when channel is full", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "out.har")
		harWriter := har.NewWriter(path, har.WithFileSystem(afero.NewOsFs()))

		for range 10_000 {
			harWriter.AddEntry(har.Entry{})
		}

		require.NoError(t, harWriter.Close())
	})

	t.Run("file is valid JSON after Close with no entries", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "empty.har")
		harWriter := har.NewWriter(path, har.WithFileSystem(afero.NewOsFs()))

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
		harWriter := har.NewWriter(path, har.WithFileSystem(afero.NewOsFs()))
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

		harWriter := har.NewWriter(path, har.WithFileSystem(afero.NewOsFs()))
		harWriter.AddEntry(har.Entry{})

		require.NoError(t, harWriter.Close())

		_, statErr := os.Stat(path)
		assert.True(t, os.IsNotExist(statErr))
	})

	t.Run("flush handles rename failure gracefully", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "out.har")

		require.NoError(t, os.Mkdir(path, 0o755))

		harWriter := har.NewWriter(path, har.WithFileSystem(afero.NewOsFs()))
		harWriter.AddEntry(har.Entry{})

		require.NoError(t, harWriter.Close())

		fi, statErr := os.Stat(path)
		require.NoError(t, statErr)
		assert.True(t, fi.IsDir())
	})

	t.Run("creates parent directories automatically", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "nested", "deep", "out.har")
		harWriter := har.NewWriter(path, har.WithFileSystem(afero.NewOsFs()))
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

	t.Run("writes the same JSON as an indented archive", func(t *testing.T) {
		started := time.Date(2026, time.October, 16, 12, 0, 0, 0, time.UTC)
		entries := []har.Entry{
			{
				StartedDateTime: started,
				Time:            12.5,
				Request:         har.Request{Method: "GET", URL: "http://example.com/a", HeadersSize: -1},
				Response:        har.Response{Status: 200, Content: har.Content{Text: "<a & b>"}},
			},
			{
				StartedDateTime: started.Add(time.Second),
				Request: har.Request{
					Method:   "POST",
					URL:      "http://example.com/b",
					PostData: &har.PostData{MimeType: "application/json", Text: `{"a":1}`},
				},
				Response: har.Response{Status: 201},
			},
		}

		for _, count := range []int{0, 1, 2} {
			t.Run(fmt.Sprintf("%d entries", count), func(t *testing.T) {
				fs := afero.NewMemMapFs()
				harWriter := har.NewWriter("/out.har", har.WithFileSystem(fs))

				for _, entry := range entries[:count] {
					require.True(t, harWriter.AddEntry(entry))
				}

				require.NoError(t, harWriter.Close())

				expected, err := json.MarshalIndent(har.HAR{
					Log: har.Log{
						Version: "1.2",
						Creator: har.Creator{Name: "uncors", Version: "dev"},
						Entries: append([]har.Entry{}, entries[:count]...),
					},
				}, "", "  ")
				require.NoError(t, err)

				actual, err := afero.ReadFile(fs, "/out.har")
				require.NoError(t, err)
				assert.Equal(t, string(expected)+"\n", string(actual))
			})
		}
	})

	t.Run("writes through the injected file system only", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "recordings", "out.har")
		fs := afero.NewMemMapFs()

		harWriter := har.NewWriter(path, har.WithFileSystem(fs))
		harWriter.AddEntry(har.Entry{Request: har.Request{Method: "GET"}})
		require.NoError(t, harWriter.Close())

		exists, err := afero.Exists(fs, path)
		require.NoError(t, err)
		assert.True(t, exists)
		assert.NoDirExists(t, filepath.Join(dir, "recordings"))
	})

	t.Run("leaves no temporary files behind", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "out.har")

		harWriter := har.NewWriter(path, har.WithFileSystem(afero.NewOsFs()))
		for range 100 {
			harWriter.AddEntry(har.Entry{Request: har.Request{Method: "GET"}})
		}

		require.NoError(t, harWriter.Close())

		files, err := os.ReadDir(dir)
		require.NoError(t, err)
		require.Len(t, files, 1)
		assert.Equal(t, "out.har", files[0].Name())

		info, err := files[0].Info()
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	})

	t.Run("requires a file system", func(t *testing.T) {
		assert.PanicsWithValue(t, "HARWriter: File system is not configured", func() {
			har.NewWriter("out.har")
		})
	})
}
