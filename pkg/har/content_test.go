package har_test

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"testing"

	"github.com/evg4b/uncors/pkg/har"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func gzipped(t *testing.T, data string) []byte {
	t.Helper()

	var buffer bytes.Buffer

	writer := gzip.NewWriter(&buffer)
	_, err := writer.Write([]byte(data))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	return buffer.Bytes()
}

func TestNewContent(t *testing.T) {
	t.Run("decodes a compressed text body", func(t *testing.T) {
		content := har.NewContent(gzipped(t, "hello"), "gzip", "text/plain")

		assert.Equal(t, har.Content{Size: 5, MimeType: "text/plain", Text: "hello"}, content)
	})

	t.Run("stores an undecodable body as base64", func(t *testing.T) {
		raw := []byte{0x1f, 0x8b, 0x00}

		content := har.NewContent(raw, "compress", "application/octet-stream")

		assert.Equal(t, har.Content{
			Size:     3,
			MimeType: "application/octet-stream",
			Text:     base64.StdEncoding.EncodeToString(raw),
			Encoding: "base64",
		}, content)
	})
}

func TestDecodeBody(t *testing.T) {
	t.Run("returns identity bodies as they are", func(t *testing.T) {
		for _, encoding := range []string{"", "identity", " Identity "} {
			decoded, err := har.DecodeBody([]byte("plain"), encoding)

			require.NoError(t, err)
			assert.Equal(t, "plain", string(decoded))
		}
	})

	t.Run("decodes gzip regardless of case", func(t *testing.T) {
		for _, encoding := range []string{"gzip", "x-gzip", "GZIP"} {
			decoded, err := har.DecodeBody(gzipped(t, "hello"), encoding)

			require.NoError(t, err)
			assert.Equal(t, "hello", string(decoded))
		}
	})

	t.Run("reports an unsupported encoding", func(t *testing.T) {
		_, err := har.DecodeBody([]byte("data"), "compress")

		require.ErrorIs(t, err, har.ErrUnsupportedEncoding)
		assert.Contains(t, err.Error(), `"compress"`)
	})

	t.Run("reports a corrupt body", func(t *testing.T) {
		_, err := har.DecodeBody([]byte("not-gzip"), "gzip")

		require.Error(t, err)
	})
}
