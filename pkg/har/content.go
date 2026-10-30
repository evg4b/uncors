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

const MaxDecodedBodySize = 50 << 20

var (
	ErrUnsupportedEncoding = errors.New("unsupported content encoding")
	ErrDecodedBodyTooLarge = fmt.Errorf("decoded body is larger than %d bytes", MaxDecodedBodySize)
)

var zstdDecoder = sync.OnceValues(func() (*zstd.Decoder, error) {
	return zstd.NewReader(nil,
		zstd.WithDecoderConcurrency(0),
		zstd.WithDecoderMaxMemory(MaxDecodedBodySize),
	)
})

func NewContent(raw []byte, contentEncoding, mimeType string) Content {
	if len(raw) == 0 {
		return Content{Size: 0, MimeType: mimeType}
	}

	decoded, err := DecodeBody(raw, contentEncoding)
	if err != nil {
		decoded = raw
	}

	content := Content{Size: int64(len(decoded)), MimeType: mimeType}
	if errors.Is(err, ErrDecodedBodyTooLarge) {
		content.Comment = "Body is stored as received: " + err.Error()
	}

	if err == nil && !isIdentity(contentEncoding) {
		compression := int64(len(decoded) - len(raw))
		content.Compression = &compression
	}

	if err != nil || !utf8.Valid(decoded) {
		content.Text = base64.StdEncoding.EncodeToString(decoded)
		content.Encoding = "base64"

		return content
	}

	content.Text = string(decoded)

	return content
}

func DecodeBody(raw []byte, contentEncoding string) ([]byte, error) {
	encoding := normalizeEncoding(contentEncoding)

	switch encoding {
	case "", "identity":
		return raw, nil
	case "gzip", "x-gzip":
		return decodeGzip(raw)
	case "deflate":
		return decodeDeflate(raw)
	case "br":
		return readLimited(brotli.NewReader(bytes.NewReader(raw)))
	case "zstd":
		return decodeZstd(raw)
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedEncoding, encoding)
	}
}

func normalizeEncoding(contentEncoding string) string {
	return strings.ToLower(strings.TrimSpace(contentEncoding))
}

func isIdentity(contentEncoding string) bool {
	encoding := normalizeEncoding(contentEncoding)

	return encoding == "" || encoding == "identity"
}

func decodeZstd(data []byte) ([]byte, error) {
	decoder, err := zstdDecoder()
	if err != nil {
		return nil, err
	}

	decoded, err := decoder.DecodeAll(data, nil)
	if errors.Is(err, zstd.ErrDecoderSizeExceeded) {
		return nil, ErrDecodedBodyTooLarge
	}

	return decoded, err
}

func decodeGzip(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	defer reader.Close()

	return readLimited(reader)
}

func decodeDeflate(data []byte) ([]byte, error) {
	// HTTP "deflate" is technically zlib-wrapped DEFLATE (RFC 1950), but many
	// servers incorrectly send raw DEFLATE (RFC 1951). Try zlib first, then
	// fall back to raw flate so both variants are handled correctly.
	decoded, err := decodeZlib(data)
	if err == nil || errors.Is(err, ErrDecodedBodyTooLarge) {
		return decoded, err
	}

	reader := flate.NewReader(bytes.NewReader(data))
	defer reader.Close()

	return readLimited(reader)
}

func decodeZlib(data []byte) ([]byte, error) {
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	defer reader.Close()

	return readLimited(reader)
}

func readLimited(reader io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(reader, MaxDecodedBodySize+1))
	if err != nil {
		return nil, err
	}

	if len(data) > MaxDecodedBodySize {
		return nil, ErrDecodedBodyTooLarge
	}

	return data, nil
}
