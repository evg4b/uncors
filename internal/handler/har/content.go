package har

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"encoding/base64"
	"io"
	"strings"
)

// buildContent creates a HAR Content object from the raw (possibly encoded)
// response body. It attempts to decode gzip and deflate-compressed bodies so
// the HAR stores readable text. Unknown or undecipherable encodings are stored
// as base64 per the HAR 1.2 spec.
func buildContent(raw []byte, contentEncoding, mimeType string) Content {
	if len(raw) == 0 {
		return Content{Size: 0, MimeType: mimeType}
	}

	encoding := strings.ToLower(strings.TrimSpace(contentEncoding))

	switch encoding {
	case "gzip", "x-gzip":
		decoded, err := decodeGzip(raw)
		if err == nil {
			return Content{
				Size:     int64(len(decoded)),
				MimeType: mimeType,
				Text:     string(decoded),
			}
		}

	case "deflate":
		decoded, err := decodeDeflate(raw)
		if err == nil {
			return Content{
				Size:     int64(len(decoded)),
				MimeType: mimeType,
				Text:     string(decoded),
			}
		}

	case "", "identity":
		// No compression — store as plain text.
		return Content{
			Size:     int64(len(raw)),
			MimeType: mimeType,
			Text:     string(raw),
		}
	}

	// Unknown or failed encoding: store as base64 so the HAR remains valid.
	return Content{
		Size:     int64(len(raw)),
		MimeType: mimeType,
		Text:     base64.StdEncoding.EncodeToString(raw),
		Encoding: "base64",
	}
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
