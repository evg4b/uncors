package config_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/go-http-utils/headers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestResponseUnmarshalYAML(t *testing.T) {
	t.Run("decodes all fields", func(t *testing.T) {
		const input = `
code: 200
headers:
  Content-Type: application/json
  X-Custom: value
delay: 200ms
raw: '{"ok":true}'
file: ./body.json
`

		var actual config.Response

		require.NoError(t, yaml.Unmarshal([]byte(input), &actual))
		assert.Equal(t, config.Response{
			Code: 200,
			Headers: map[string]string{
				"Content-Type": "application/json",
				"X-Custom":     "value",
			},
			Delay: 200 * time.Millisecond,
			Raw:   `{"ok":true}`,
			File:  "./body.json",
		}, actual)
	})

	t.Run("zero delay when field is absent", func(t *testing.T) {
		const input = `code: 204`

		var actual config.Response

		require.NoError(t, yaml.Unmarshal([]byte(input), &actual))
		assert.Zero(t, actual.Delay)
	})

	t.Run("parses delay with embedded spaces", func(t *testing.T) {
		const input = `delay: "1s500ms"`

		var actual config.Response

		require.NoError(t, yaml.Unmarshal([]byte(input), &actual))
		assert.Equal(t, 1500*time.Millisecond, actual.Delay)
	})

	t.Run("returns error for invalid delay", func(t *testing.T) {
		const input = `delay: not-a-duration`

		var actual config.Response

		assert.Error(t, yaml.Unmarshal([]byte(input), &actual))
	})
}

func TestResponseClone(t *testing.T) {
	response := config.Response{
		Code: http.StatusOK,
		Headers: map[string]string{
			headers.ContentType:  "plain/text",
			headers.CacheControl: "none",
		},
		Raw:   "this is plain text",
		File:  "~/projects/uncors/response/demo.json",
		Delay: time.Hour,
	}

	clonedResponse := response.Clone()

	t.Run("not same", func(t *testing.T) {
		assert.NotSame(t, &response, &clonedResponse)
	})

	t.Run("equals values", func(t *testing.T) {
		assert.Equal(t, response, clonedResponse)
	})

	t.Run("not same Headers map", func(t *testing.T) {
		assert.NotSame(t, &response.Headers, &clonedResponse.Headers)
	})

	t.Run("response type", func(t *testing.T) {
		t.Run("raw response", func(t *testing.T) {
			response := config.Response{
				Code: http.StatusOK,
				Raw:  "this is plain text",
			}

			t.Run("IsRaw", func(t *testing.T) {
				assert.True(t, response.IsRaw())
			})

			t.Run("IsFile", func(t *testing.T) {
				assert.False(t, response.IsFile())
			})
		})

		t.Run("file response", func(t *testing.T) {
			response := config.Response{
				Code: http.StatusOK,
				File: "~/projects/uncors/response/demo.json",
			}

			t.Run("IsRaw", func(t *testing.T) {
				assert.False(t, response.IsRaw())
			})

			t.Run("IsFile", func(t *testing.T) {
				assert.True(t, response.IsFile())
			})
		})
	})
}
