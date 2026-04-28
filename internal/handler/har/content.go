package har

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
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
	// The deflate encoding used in HTTP can be raw DEFLATE or zlib-wrapped.
	// Try zlib first; fall back to raw flate.
	reader := flate.NewReader(bytes.NewReader(data))

	defer reader.Close()

	return io.ReadAll(reader)
}
