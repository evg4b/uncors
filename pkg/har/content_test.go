package har_test

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"encoding/base64"
	"io"
	"testing"

	"github.com/andybalholm/brotli"
	"github.com/evg4b/uncors/pkg/har"
	"github.com/klauspost/compress/zstd"
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

func compressed(t *testing.T, size int, open func(io.Writer) io.WriteCloser) []byte {
	t.Helper()

	var buffer bytes.Buffer

	writer := open(&buffer)
	_, err := writer.Write(make([]byte, size))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	return buffer.Bytes()
}

var compressors = map[string]func(io.Writer) io.WriteCloser{
	"gzip": func(w io.Writer) io.WriteCloser {
		writer, _ := gzip.NewWriterLevel(w, gzip.BestSpeed)

		return writer
	},
	"deflate": func(w io.Writer) io.WriteCloser {
		writer, _ := zlib.NewWriterLevel(w, zlib.BestSpeed)

		return writer
	},
	"br": func(w io.Writer) io.WriteCloser {
		return brotli.NewWriterLevel(w, brotli.BestSpeed)
	},
	"zstd": func(w io.Writer) io.WriteCloser {
		encoder, _ := zstd.NewWriter(w, zstd.WithEncoderLevel(zstd.SpeedFastest))

		return encoder
	},
}

func TestNewContent(t *testing.T) {
	t.Run("decodes a compressed text body", func(t *testing.T) {
		content := har.NewContent(gzipped(t, "hello"), "gzip", "text/plain")

		compression := int64(5 - len(gzipped(t, "hello")))
		assert.Equal(t, har.Content{Size: 5, Compression: &compression, MimeType: "text/plain", Text: "hello"}, content)
	})

	t.Run("leaves out compression for an uncompressed body", func(t *testing.T) {
		content := har.NewContent([]byte("hello"), "identity", "text/plain")

		assert.Nil(t, content.Compression)
	})

	t.Run("stores a body that decodes past the size limit as received", func(t *testing.T) {
		raw := compressed(t, har.MaxDecodedBodySize+1, compressors["gzip"])

		content := har.NewContent(raw, "gzip", "application/json")

		assert.Equal(t, int64(len(raw)), content.Size)
		assert.Nil(t, content.Compression)
		assert.Equal(t, "base64", content.Encoding)
		assert.Contains(t, content.Comment, "decoded body is larger than")
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
			decoded, err := har.DecodeBody(gzipped(t, "gzip body"), encoding)

			require.NoError(t, err)
			assert.Equal(t, "gzip body", string(decoded))
		}
	})

	t.Run("reports an unsupported encoding", func(t *testing.T) {
		_, err := har.DecodeBody([]byte("data"), "compress")

		require.ErrorIs(t, err, har.ErrUnsupportedEncoding)
		assert.Contains(t, err.Error(), `"compress"`)
	})

	t.Run("decodes a body at the size limit", func(t *testing.T) {
		for _, encoding := range []string{"gzip", "zstd"} {
			decoded, err := har.DecodeBody(compressed(t, har.MaxDecodedBodySize, compressors[encoding]), encoding)

			require.NoError(t, err, encoding)
			assert.Len(t, decoded, har.MaxDecodedBodySize, encoding)
		}
	})

	t.Run("refuses a body that decodes past the size limit", func(t *testing.T) {
		for encoding, open := range compressors {
			_, err := har.DecodeBody(compressed(t, har.MaxDecodedBodySize+1, open), encoding)

			require.ErrorIs(t, err, har.ErrDecodedBodyTooLarge, encoding)
		}
	})

	t.Run("refuses a raw deflate body past the size limit", func(t *testing.T) {
		raw := compressed(t, har.MaxDecodedBodySize+1, func(w io.Writer) io.WriteCloser {
			writer, _ := flate.NewWriter(w, flate.BestSpeed)

			return writer
		})

		_, err := har.DecodeBody(raw, "deflate")

		require.ErrorIs(t, err, har.ErrDecodedBodyTooLarge)
	})

	t.Run("reports a corrupt body", func(t *testing.T) {
		_, err := har.DecodeBody([]byte("not-gzip"), "gzip")

		require.Error(t, err)
	})
}
