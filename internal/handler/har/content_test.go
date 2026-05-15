package har_test

import (
	"encoding/base64"
	"testing"

	"github.com/evg4b/uncors/internal/handler/har"
	"github.com/stretchr/testify/assert"
)

func TestContent_Encoding(t *testing.T) {
	t.Run("unknown encoding stored as base64", func(t *testing.T) {
		raw := []byte{0x1f, 0x8b, 0x00} // truncated gzip → cannot decode
		b64 := base64.StdEncoding.EncodeToString(raw)

		entry := har.Entry{
			Response: har.Response{
				Content: har.Content{
					Text:     b64,
					Encoding: "base64",
					MimeType: "application/octet-stream",
				},
			},
		}

		assert.Equal(t, "base64", entry.Response.Content.Encoding)
		assert.Equal(t, b64, entry.Response.Content.Text)
	})
}
