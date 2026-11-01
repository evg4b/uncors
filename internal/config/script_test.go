package config_test

import (
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/headers"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestScriptUnmarshalYAML(t *testing.T) {
	t.Run("decodes limits", func(t *testing.T) {
		const input = `
path: /api/slow
script: response:WriteString("ok")
timeout: 2m30s
max-body-size: 52428800
`

		var actual config.Script

		require.NoError(t, yaml.Unmarshal([]byte(input), &actual))
		assert.Equal(t, 150*time.Second, actual.Timeout)
		assert.Equal(t, int64(52428800), actual.MaxBodySize)
	})

	t.Run("zero limits when fields are absent", func(t *testing.T) {
		const input = `script: response:WriteString("ok")`

		var actual config.Script

		require.NoError(t, yaml.Unmarshal([]byte(input), &actual))
		assert.Zero(t, actual.Timeout)
		assert.Zero(t, actual.MaxBodySize)
	})

	t.Run("returns error for invalid max-body-size", func(t *testing.T) {
		const input = `max-body-size: 10MB`

		var actual config.Script

		assert.Error(t, yaml.Unmarshal([]byte(input), &actual))
	})

	t.Run("returns error for invalid timeout", func(t *testing.T) {
		const input = `timeout: not-a-duration`

		var actual config.Script

		assert.Error(t, yaml.Unmarshal([]byte(input), &actual))
	})
}

func TestScript_String(t *testing.T) {
	tests := []struct {
		name     string
		script   config.Script
		expected string
	}{
		{
			name: "inline script with method",
			script: config.Script{
				Matcher: config.RequestMatcher{
					Path:   "/api/test",
					Method: "POST",
				},
				Script: "response:WriteString('hello')",
			},
			expected: "[POST script:inline] /api/test",
		},
		{
			name: "file script with method",
			script: config.Script{
				Matcher: config.RequestMatcher{
					Path:   "/api/handler",
					Method: "GET",
				},
				File: "/scripts/handler.lua",
			},
			expected: "[GET script:file: /scripts/handler.lua] /api/handler",
		},
		{
			name: "inline script without method",
			script: config.Script{
				Matcher: config.RequestMatcher{
					Path: "/api/wildcard",
				},
				Script: "response:WriteString('any method')",
			},
			expected: "[* script:inline] /api/wildcard",
		},
		{
			name: "file script without method",
			script: config.Script{
				Matcher: config.RequestMatcher{
					Path: "/api/any",
				},
				File: "/scripts/any.lua",
			},
			expected: "[* script:file: /scripts/any.lua] /api/any",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.script.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

const (
	testAPIPath        = "/api/test"
	testScriptContent  = "response.status = 200"
	testScriptFilePath = "/scripts/test.lua"
	scriptPathField    = "script.path"
	scriptScriptField  = "script.script"
	scriptFileField    = "script.file"
)

func TestScriptValidator(t *testing.T) {
	noFS := testutils.FsFromMap(t, map[string]string{})

	t.Run("valid inline script", func(t *testing.T) {
		err := (&config.Script{
			Matcher: config.RequestMatcher{Path: testAPIPath, Method: "GET"},
			Script:  testScriptContent,
		}).Validate("script", noFS)
		assert.NoError(t, err)
	})

	t.Run("valid file script", func(t *testing.T) {
		fs := testutils.FsFromMap(t, map[string]string{testScriptFilePath: testScriptContent})

		err := (&config.Script{
			Matcher: config.RequestMatcher{Path: testAPIPath, Method: "POST"},
			File:    testScriptFilePath,
		}).Validate("script", fs)
		assert.NoError(t, err)
	})

	t.Run("empty method is allowed", func(t *testing.T) {
		err := (&config.Script{
			Matcher: config.RequestMatcher{Path: testAPIPath},
			Script:  testScriptContent,
		}).Validate("script", noFS)
		assert.NoError(t, err)
	})

	t.Run("valid queries and headers", func(t *testing.T) {
		err := (&config.Script{
			Matcher: config.RequestMatcher{
				Path:    "/api/test",
				Queries: map[string]string{"filter": "active"},
				Headers: map[string]string{headers.Authorization: "Bearer token"},
			},
			Script: testScriptContent,
		}).Validate("script", noFS)
		assert.NoError(t, err)
	})

	t.Run("empty path", func(t *testing.T) {
		err := (&config.Script{
			Matcher: config.RequestMatcher{Path: ""},
			Script:  testScriptContent,
		}).Validate("script", noFS)
		require.Error(t, err)
		assert.Contains(t, err.Error(), scriptPathField)
	})

	t.Run("invalid path", func(t *testing.T) {
		err := (&config.Script{
			Matcher: config.RequestMatcher{Path: "invalid-path"},
			Script:  testScriptContent,
		}).Validate("script", noFS)
		require.Error(t, err)
		assert.Contains(t, err.Error(), scriptPathField)
	})

	t.Run("invalid method", func(t *testing.T) {
		err := (&config.Script{
			Matcher: config.RequestMatcher{Path: testAPIPath, Method: "INVALID"},
			Script:  testScriptContent,
		}).Validate("script", noFS)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "script.method")
	})

	t.Run("neither script nor file provided", func(t *testing.T) {
		err := (&config.Script{
			Matcher: config.RequestMatcher{Path: testAPIPath, Method: "GET"},
		}).Validate("script", noFS)
		require.Error(t, err)
		assert.Contains(t, err.Error(), scriptScriptField)
		assert.Contains(t, err.Error(), scriptFileField)
		assert.Contains(t, err.Error(), "either 'script' or 'file' must be provided")
	})

	t.Run("both script and file provided", func(t *testing.T) {
		fs := testutils.FsFromMap(t, map[string]string{testScriptFilePath: testScriptContent})

		err := (&config.Script{
			Matcher: config.RequestMatcher{Path: "/api/test"},
			Script:  testScriptContent,
			File:    "/scripts/test.lua",
		}).Validate("script", fs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), scriptScriptField)
		assert.Contains(t, err.Error(), scriptFileField)
		assert.Contains(t, err.Error(), "only one of 'script' or 'file' can be provided")
	})

	t.Run("file does not exist", func(t *testing.T) {
		err := (&config.Script{
			Matcher: config.RequestMatcher{Path: "/api/test"},
			File:    "/scripts/nonexistent.lua",
		}).Validate("script", noFS)
		require.Error(t, err)
		assert.Contains(t, err.Error(), scriptFileField)
	})

	t.Run("valid timeout", func(t *testing.T) {
		err := (&config.Script{
			Matcher: config.RequestMatcher{Path: testAPIPath},
			Script:  testScriptContent,
			Timeout: 2 * time.Minute,
		}).Validate("script", noFS)
		assert.NoError(t, err)
	})

	t.Run("negative timeout", func(t *testing.T) {
		err := (&config.Script{
			Matcher: config.RequestMatcher{Path: testAPIPath},
			Script:  testScriptContent,
			Timeout: -time.Second,
		}).Validate("script", noFS)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "script.timeout must be greater than or equal to 0")
	})

	t.Run("valid max-body-size", func(t *testing.T) {
		err := (&config.Script{
			Matcher:     config.RequestMatcher{Path: testAPIPath},
			Script:      testScriptContent,
			MaxBodySize: 1024,
		}).Validate("script", noFS)
		assert.NoError(t, err)
	})

	t.Run("negative max-body-size", func(t *testing.T) {
		err := (&config.Script{
			Matcher:     config.RequestMatcher{Path: testAPIPath},
			Script:      testScriptContent,
			MaxBodySize: -1,
		}).Validate("script", noFS)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "script.max-body-size must be greater than or equal to 0")
	})

	t.Run("multiple errors", func(t *testing.T) {
		err := (&config.Script{
			Matcher: config.RequestMatcher{Path: "", Method: "INVALID"},
		}).Validate("script", noFS)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "script.path")
		assert.Contains(t, err.Error(), "script.method")
		assert.Contains(t, err.Error(), "script.script")
		assert.Contains(t, err.Error(), "script.file")
	})
}
