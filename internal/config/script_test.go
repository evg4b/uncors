package config_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/go-http-utils/headers"
	"github.com/stretchr/testify/assert"
)

func TestRequestMatcher_Clone(t *testing.T) {
	original := config.RequestMatcher{
		Path:   "/api/test",
		Method: "POST",
		Queries: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
		Headers: map[string]string{
			headers.ContentType:   "application/json",
			headers.Authorization: "Bearer token",
		},
	}

	cloned := original.Clone()

	assert.Equal(t, original.Path, cloned.Path)
	assert.Equal(t, original.Method, cloned.Method)
	assert.Equal(t, original.Queries, cloned.Queries)
	assert.Equal(t, original.Headers, cloned.Headers)

	// Verify deep copy
	cloned.Queries["key1"] = "modified"
	assert.NotEqual(t, original.Queries["key1"], cloned.Queries["key1"])

	cloned.Headers[headers.ContentType] = "text/html"
	assert.NotEqual(t, original.Headers[headers.ContentType], cloned.Headers[headers.ContentType])
}

func TestScript_Clone(t *testing.T) {
	original := config.Script{
		Matcher: config.RequestMatcher{
			Path:   "/api/script",
			Method: "GET",
			Queries: map[string]string{
				"param": "value",
			},
			Headers: map[string]string{
				"X-Custom": "header",
			},
		},
		Script: "print('hello')",
		File:   "/path/to/script.lua",
	}

	cloned := original.Clone()

	assert.Equal(t, original.Matcher.Path, cloned.Matcher.Path)
	assert.Equal(t, original.Matcher.Method, cloned.Matcher.Method)
	assert.Equal(t, original.Script, cloned.Script)
	assert.Equal(t, original.File, cloned.File)
	assert.Equal(t, original.Matcher.Queries, cloned.Matcher.Queries)
	assert.Equal(t, original.Matcher.Headers, cloned.Matcher.Headers)

	// Verify deep copy
	cloned.Matcher.Queries["param"] = "modified"
	assert.NotEqual(t, original.Matcher.Queries["param"], cloned.Matcher.Queries["param"])
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

func TestScripts_Clone(t *testing.T) {
	t.Run("non-nil scripts", func(t *testing.T) {
		original := config.Scripts{
			{
				Matcher: config.RequestMatcher{
					Path:   "/api/one",
					Method: "GET",
				},
				Script: "script1",
			},
			{
				Matcher: config.RequestMatcher{
					Path:   "/api/two",
					Method: "POST",
				},
				File: "/scripts/two.lua",
			},
		}

		cloned := original.Clone()

		assert.Len(t, cloned, len(original))
		assert.Equal(t, original[0].Matcher.Path, cloned[0].Matcher.Path)
		assert.Equal(t, original[1].Matcher.Path, cloned[1].Matcher.Path)

		// Verify deep copy
		cloned[0].Matcher.Path = "/modified"
		assert.NotEqual(t, original[0].Matcher.Path, cloned[0].Matcher.Path)
	})

	t.Run("nil scripts", func(t *testing.T) {
		var original config.Scripts

		cloned := original.Clone()
		assert.Nil(t, cloned)
	})

	t.Run("empty scripts", func(t *testing.T) {
		original := config.Scripts{}
		cloned := original.Clone()
		assert.Empty(t, cloned)
		assert.NotNil(t, cloned)
	})
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
		var errs config.Errors
		(&config.Script{
			Matcher: config.RequestMatcher{Path: testAPIPath, Method: "GET"},
			Script:  testScriptContent,
		}).Validate("script", noFS, &errs)
		assert.False(t, errs.HasAny())
	})

	t.Run("valid file script", func(t *testing.T) {
		fs := testutils.FsFromMap(t, map[string]string{testScriptFilePath: testScriptContent})

		var errs config.Errors
		(&config.Script{
			Matcher: config.RequestMatcher{Path: testAPIPath, Method: "POST"},
			File:    testScriptFilePath,
		}).Validate("script", fs, &errs)
		assert.False(t, errs.HasAny())
	})

	t.Run("empty method is allowed", func(t *testing.T) {
		var errs config.Errors
		(&config.Script{
			Matcher: config.RequestMatcher{Path: testAPIPath},
			Script:  testScriptContent,
		}).Validate("script", noFS, &errs)
		assert.False(t, errs.HasAny())
	})

	t.Run("valid queries and headers", func(t *testing.T) {
		var errs config.Errors
		(&config.Script{
			Matcher: config.RequestMatcher{
				Path:    "/api/test",
				Queries: map[string]string{"filter": "active"},
				Headers: map[string]string{headers.Authorization: "Bearer token"},
			},
			Script: testScriptContent,
		}).Validate("script", noFS, &errs)
		assert.False(t, errs.HasAny())
	})

	t.Run("empty path", func(t *testing.T) {
		var errs config.Errors
		(&config.Script{
			Matcher: config.RequestMatcher{Path: ""},
			Script:  testScriptContent,
		}).Validate("script", noFS, &errs)
		assert.True(t, errs.HasAny())
		assert.Contains(t, errs.Error(), scriptPathField)
	})

	t.Run("invalid path", func(t *testing.T) {
		var errs config.Errors
		(&config.Script{
			Matcher: config.RequestMatcher{Path: "invalid-path"},
			Script:  testScriptContent,
		}).Validate("script", noFS, &errs)
		assert.True(t, errs.HasAny())
		assert.Contains(t, errs.Error(), scriptPathField)
	})

	t.Run("invalid method", func(t *testing.T) {
		var errs config.Errors
		(&config.Script{
			Matcher: config.RequestMatcher{Path: testAPIPath, Method: "INVALID"},
			Script:  testScriptContent,
		}).Validate("script", noFS, &errs)
		assert.True(t, errs.HasAny())
		assert.Contains(t, errs.Error(), "script.method")
	})

	t.Run("neither script nor file provided", func(t *testing.T) {
		var errs config.Errors
		(&config.Script{
			Matcher: config.RequestMatcher{Path: testAPIPath, Method: "GET"},
		}).Validate("script", noFS, &errs)
		assert.True(t, errs.HasAny())
		assert.Contains(t, errs.Error(), scriptScriptField)
		assert.Contains(t, errs.Error(), scriptFileField)
		assert.Contains(t, errs.Error(), "either 'script' or 'file' must be provided")
	})

	t.Run("both script and file provided", func(t *testing.T) {
		fs := testutils.FsFromMap(t, map[string]string{testScriptFilePath: testScriptContent})

		var errs config.Errors
		(&config.Script{
			Matcher: config.RequestMatcher{Path: "/api/test"},
			Script:  testScriptContent,
			File:    "/scripts/test.lua",
		}).Validate("script", fs, &errs)
		assert.True(t, errs.HasAny())
		assert.Contains(t, errs.Error(), scriptScriptField)
		assert.Contains(t, errs.Error(), scriptFileField)
		assert.Contains(t, errs.Error(), "only one of 'script' or 'file' can be provided")
	})

	t.Run("file does not exist", func(t *testing.T) {
		var errs config.Errors
		(&config.Script{
			Matcher: config.RequestMatcher{Path: "/api/test"},
			File:    "/scripts/nonexistent.lua",
		}).Validate("script", noFS, &errs)
		assert.True(t, errs.HasAny())
		assert.Contains(t, errs.Error(), scriptFileField)
	})

	t.Run("multiple errors", func(t *testing.T) {
		var errs config.Errors
		(&config.Script{
			Matcher: config.RequestMatcher{Path: "", Method: "INVALID"},
		}).Validate("script", noFS, &errs)
		assert.True(t, errs.HasAny())
		errStr := errs.Error()
		assert.Contains(t, errStr, "script.path")
		assert.Contains(t, errStr, "script.method")
		assert.Contains(t, errStr, "script.script")
		assert.Contains(t, errStr, "script.file")
	})
}
