package har

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
)

var errUnsupportedEncoding = errors.New("unsupported content encoding")

var zstdDecoder = sync.OnceValues(func() (*zstd.Decoder, error) {
	return zstd.NewReader(nil, zstd.WithDecoderConcurrency(0))
})

func buildContent(raw []byte, contentEncoding, mimeType string) Content {
	if len(raw) == 0 {
		return Content{Size: 0, MimeType: mimeType}
	}

	decoded, err := decodeBody(raw, strings.ToLower(strings.TrimSpace(contentEncoding)))
	if err != nil {
		decoded = raw
	}

	content := Content{Size: int64(len(decoded)), MimeType: mimeType}
	if err != nil || !utf8.Valid(decoded) {
		content.Text = base64.StdEncoding.EncodeToString(decoded)
		content.Encoding = "base64"

		return content
	}

	content.Text = string(decoded)

	return content
}

func decodeBody(raw []byte, encoding string) ([]byte, error) {
	switch encoding {
	case "", "identity":
		return raw, nil
	case "gzip", "x-gzip":
		return decodeGzip(raw)
	case "deflate":
		return decodeDeflate(raw)
	case "br":
		return io.ReadAll(brotli.NewReader(bytes.NewReader(raw)))
	case "zstd":
		return decodeZstd(raw)
	default:
		return nil, fmt.Errorf("%w: %q", errUnsupportedEncoding, encoding)
	}
}

func decodeZstd(data []byte) ([]byte, error) {
	decoder, err := zstdDecoder()
	if err != nil {
		return nil, err
	}

	return decoder.DecodeAll(data, nil)
}

func decodeGzip(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	defer reader.Close()

	return io.ReadAll(reader)
}

func decodeDeflate(data []byte) ([]byte, error) {
	// HTTP "deflate" is technically zlib-wrapped DEFLATE (RFC 1950), but many
	// servers incorrectly send raw DEFLATE (RFC 1951). Try zlib first, then
	// fall back to raw flate so both variants are handled correctly.
	decoded, err := decodeZlib(data)
	if err == nil {
		return decoded, nil
	}

	reader := flate.NewReader(bytes.NewReader(data))
	defer reader.Close()

	return io.ReadAll(reader)
}

func decodeZlib(data []byte) ([]byte, error) {
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	defer reader.Close()

	return io.ReadAll(reader)
}
