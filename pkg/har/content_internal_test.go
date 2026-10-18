package har

import (
	"bytes"
	"compress/flate"
	"encoding/base64"
	"testing"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
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

func compressBrotli(t *testing.T, data []byte) []byte {
	t.Helper()

	var buf bytes.Buffer

	w := brotli.NewWriter(&buf)
	_, err := w.Write(data)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	return buf.Bytes()
}

func compressZstd(t *testing.T, data []byte) []byte {
	t.Helper()

	encoder, err := zstd.NewWriter(nil)
	require.NoError(t, err)

	defer encoder.Close()

	return encoder.EncodeAll(data, nil)
}

func TestBuildContent(t *testing.T) {
	t.Run("empty body returns zero-size content", func(t *testing.T) {
		c := NewContent(nil, "", "text/plain")

		assert.Equal(t, int64(0), c.Size)
		assert.Equal(t, "text/plain", c.MimeType)
		assert.Empty(t, c.Text)
		assert.Empty(t, c.Encoding)
	})

	t.Run("empty encoding stores plain text", func(t *testing.T) {
		c := NewContent([]byte("hello"), "", "text/plain")

		assert.Equal(t, "hello", c.Text)
		assert.Empty(t, c.Encoding)
	})

	t.Run("identity encoding stores plain text", func(t *testing.T) {
		c := NewContent([]byte("hello"), "identity", "text/plain")

		assert.Equal(t, "hello", c.Text)
		assert.Empty(t, c.Encoding)
	})

	t.Run("unknown encoding falls back to base64", func(t *testing.T) {
		c := NewContent([]byte("data"), "compress", "text/plain")

		assert.Equal(t, "base64", c.Encoding)
		assert.Equal(t, base64.StdEncoding.EncodeToString([]byte("data")), c.Text)
		assert.Equal(t, int64(4), c.Size)
	})

	t.Run("identity body that is not UTF-8 is stored as base64", func(t *testing.T) {
		raw := []byte{0x89, 'P', 'N', 'G', 0xff, 0x00}

		c := NewContent(raw, "", "image/png")

		assert.Equal(t, "base64", c.Encoding)
		assert.Equal(t, base64.StdEncoding.EncodeToString(raw), c.Text)
		assert.Equal(t, int64(len(raw)), c.Size)
	})

	t.Run("brotli body is decoded", func(t *testing.T) {
		c := NewContent(compressBrotli(t, []byte("brotli content")), "br", "text/plain")

		assert.Equal(t, "brotli content", c.Text)
		assert.Empty(t, c.Encoding)
		assert.Equal(t, int64(len("brotli content")), c.Size)
	})

	t.Run("zstd body is decoded", func(t *testing.T) {
		c := NewContent(compressZstd(t, []byte("zstd content")), "zstd", "text/plain")

		assert.Equal(t, "zstd content", c.Text)
		assert.Empty(t, c.Encoding)
		assert.Equal(t, int64(len("zstd content")), c.Size)
	})

	t.Run("encoding name is case and space insensitive", func(t *testing.T) {
		c := NewContent(compressBrotli(t, []byte("brotli content")), " BR ", "text/plain")

		assert.Equal(t, "brotli content", c.Text)
	})

	for _, encoding := range []string{"br", "zstd"} {
		t.Run("invalid "+encoding+" data falls back to base64 of the raw bytes", func(t *testing.T) {
			raw := []byte{0xde, 0xad, 0xbe, 0xef, 0x00, 0x01}

			c := NewContent(raw, encoding, "application/octet-stream")

			assert.Equal(t, "base64", c.Encoding)
			assert.Equal(t, base64.StdEncoding.EncodeToString(raw), c.Text)
			assert.Equal(t, int64(len(raw)), c.Size)
		})
	}

	t.Run("decoded body that is not UTF-8 is stored as base64 of the decoded bytes", func(t *testing.T) {
		decoded := []byte{0xff, 0xfe, 0x00, 0x80}

		c := NewContent(compressBrotli(t, decoded), "br", "application/protobuf")

		assert.Equal(t, "base64", c.Encoding)
		assert.Equal(t, base64.StdEncoding.EncodeToString(decoded), c.Text)
		assert.Equal(t, int64(len(decoded)), c.Size)
	})

	t.Run("invalid gzip data falls back to base64", func(t *testing.T) {
		c := NewContent([]byte("not-gzip-data"), "gzip", "application/octet-stream")

		assert.Equal(t, "base64", c.Encoding)
	})

	t.Run("invalid deflate data falls back to base64", func(t *testing.T) {
		c := NewContent([]byte{0x00, 0x01, 0x02}, "deflate", "application/octet-stream")

		assert.Equal(t, "base64", c.Encoding)
	})

	t.Run("raw deflate (RFC 1951) is decoded when zlib wrapper absent", func(t *testing.T) {
		const original = "raw deflate content"

		compressed := rawDeflate(t, []byte(original))

		c := NewContent(compressed, "deflate", "text/plain")

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
