package config_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/testutils"
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

func TestResponseValidator(t *testing.T) {
	const file = "testdata/file.txt"

	fs := testutils.FsFromMap(t, map[string]string{file: "test"})

	t.Run("should not register errors if response is valid", func(t *testing.T) {
		tests := []struct {
			name  string
			value config.Response
		}{
			{name: "with file", value: config.Response{Code: 200, File: file, Delay: 3 * time.Second}},
			{name: "with raw", value: config.Response{Code: 200, Raw: `{ "test": "test" }`, Delay: 3 * time.Second}},
			{name: "without delay", value: config.Response{Code: 200, Raw: `{ "test": "test" }`}},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				var errs config.Errors
				test.value.Validate("test", fs, &errs)
				assert.False(t, errs.HasAny())
			})
		}
	})

	t.Run("should register errors for", func(t *testing.T) {
		tests := []struct {
			name  string
			value config.Response
			error string
		}{
			{
				name:  "code",
				value: config.Response{Code: 0, File: file, Delay: 3 * time.Second},
				error: "test.code code must be in range 100-599",
			},
			{
				name:  "file",
				value: config.Response{Code: 200, File: "testdata/unknown.txt", Delay: 3 * time.Second},
				error: "test.file testdata/unknown.txt does not exist",
			},
			{
				name:  "delay",
				value: config.Response{Code: 200, File: file, Delay: -1 * time.Second},
				error: "test.delay must be greater than or equal to 0",
			},
			{
				name:  "both empty",
				value: config.Response{Code: 200, Delay: 3 * time.Second},
				error: "test.raw or test.file must be set",
			},
			{
				name:  "both set",
				value: config.Response{Code: 200, File: file, Raw: "test", Delay: 3 * time.Second},
				error: "only one of test.raw or test.file must be set",
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				var errs config.Errors
				test.value.Validate("test", fs, &errs)
				require.EqualError(t, errs, test.error)
			})
		}
	})
}
