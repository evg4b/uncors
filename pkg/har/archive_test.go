package har_test

import (
	"bytes"
	"errors"
	"math"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/evg4b/uncors/pkg/har"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	creator       = har.Creator{Name: "test", Version: "1.0"}
	errDiskFull   = errors.New("disk full")
	errBrokenPipe = errors.New("broken pipe")
)

func TestNew(t *testing.T) {
	t.Run("without entries", func(t *testing.T) {
		archive := har.New(creator)

		assert.Equal(t, har.Version, archive.Log.Version)
		assert.Equal(t, creator, archive.Log.Creator)
		assert.NotNil(t, archive.Log.Entries)
		assert.Empty(t, archive.Log.Entries)
	})

	t.Run("with entries", func(t *testing.T) {
		entries := []har.Entry{
			{Request: har.Request{Method: "GET"}},
			{Request: har.Request{Method: "POST"}},
		}

		archive := har.New(creator, entries...)

		assert.Equal(t, entries, archive.Log.Entries)
	})
}

func TestEncode(t *testing.T) {
	t.Run("writes indented JSON with a trailing newline", func(t *testing.T) {
		var buffer bytes.Buffer

		require.NoError(t, har.Encode(&buffer, har.New(creator)))

		expected := "{\n" +
			"  \"log\": {\n" +
			"    \"version\": \"1.2\",\n" +
			"    \"creator\": {\n" +
			"      \"name\": \"test\",\n" +
			"      \"version\": \"1.0\"\n" +
			"    },\n" +
			"    \"entries\": []\n" +
			"  }\n" +
			"}\n"
		assert.Equal(t, expected, buffer.String())
	})

	t.Run("reports values JSON cannot hold", func(t *testing.T) {
		archive := har.New(creator, har.Entry{Time: math.NaN()})

		err := har.Encode(&bytes.Buffer{}, archive)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "har: cannot encode archive")
	})

	t.Run("reports write errors", func(t *testing.T) {
		err := har.Encode(failingWriter{err: errDiskFull}, har.New(creator))

		require.ErrorIs(t, err, errDiskFull)
	})
}

func TestDecode(t *testing.T) {
	t.Run("reads what Encode writes", func(t *testing.T) {
		started := time.Date(2026, time.October, 16, 12, 0, 0, 0, time.UTC)
		archive := har.New(creator, har.Entry{
			StartedDateTime: started,
			Time:            12.5,
			Request: har.Request{
				Method:   "POST",
				URL:      "http://example.com/api",
				Headers:  []har.NameValue{{Name: "Content-Type", Value: "application/json"}},
				PostData: &har.PostData{MimeType: "application/json", Text: `{"a":1}`},
			},
			Response: har.Response{
				Status:  201,
				Content: har.Content{Size: 2, MimeType: "text/plain", Text: "ok"},
			},
			Timings: har.Timings{Wait: 12.5},
		})

		var buffer bytes.Buffer

		require.NoError(t, har.Encode(&buffer, archive))

		decoded, err := har.Decode(&buffer)
		require.NoError(t, err)
		assert.Equal(t, archive, decoded)
	})

	t.Run("reports invalid JSON", func(t *testing.T) {
		_, err := har.Decode(strings.NewReader("{"))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "har: cannot decode archive")
	})

	t.Run("reports read errors", func(t *testing.T) {
		_, err := har.Decode(iotest.ErrReader(errBrokenPipe))

		require.ErrorIs(t, err, errBrokenPipe)
	})
}

type failingWriter struct {
	err error
}

func (w failingWriter) Write([]byte) (int, error) {
	return 0, w.err
}
