package har

import (
	"bytes"
	"compress/flate"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rawDeflate(t *testing.T, data []byte) []byte {
	t.Helper()

	var buf bytes.Buffer

	w, err := flate.NewWriter(&buf, flate.DefaultCompression)
	require.NoError(t, err)

	_, err = w.Write(data)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	return buf.Bytes()
}

func TestBuildContent(t *testing.T) {
	t.Run("empty body returns zero-size content", func(t *testing.T) {
		c := buildContent(nil, "", "text/plain")

		assert.Equal(t, int64(0), c.Size)
		assert.Equal(t, "text/plain", c.MimeType)
		assert.Empty(t, c.Text)
		assert.Empty(t, c.Encoding)
	})

	t.Run("empty encoding stores plain text", func(t *testing.T) {
		c := buildContent([]byte("hello"), "", "text/plain")

		assert.Equal(t, "hello", c.Text)
		assert.Empty(t, c.Encoding)
	})

	t.Run("identity encoding stores plain text", func(t *testing.T) {
		c := buildContent([]byte("hello"), "identity", "text/plain")

		assert.Equal(t, "hello", c.Text)
		assert.Empty(t, c.Encoding)
	})

	t.Run("unknown encoding falls back to base64", func(t *testing.T) {
		c := buildContent([]byte("data"), "br", "text/plain")

		assert.Equal(t, "base64", c.Encoding)
	})

	t.Run("invalid gzip data falls back to base64", func(t *testing.T) {
		c := buildContent([]byte("not-gzip-data"), "gzip", "application/octet-stream")

		assert.Equal(t, "base64", c.Encoding)
	})

	t.Run("invalid deflate data falls back to base64", func(t *testing.T) {
		c := buildContent([]byte{0x00, 0x01, 0x02}, "deflate", "application/octet-stream")

		assert.Equal(t, "base64", c.Encoding)
	})

	t.Run("raw deflate (RFC 1951) is decoded when zlib wrapper absent", func(t *testing.T) {
		const original = "raw deflate content"

		compressed := rawDeflate(t, []byte(original))

		c := buildContent(compressed, "deflate", "text/plain")

		assert.Equal(t, original, c.Text)
		assert.Empty(t, c.Encoding, "decoded content must not carry an encoding field")
	})
}

func TestDecodeGzip_InvalidData(t *testing.T) {
	_, err := decodeGzip([]byte("not-gzip"))

	assert.Error(t, err)
}

func TestDecodeZlib_InvalidData(t *testing.T) {
	_, err := decodeZlib([]byte("not-zlib"))

	assert.Error(t, err)
}
