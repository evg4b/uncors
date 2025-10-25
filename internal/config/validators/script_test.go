package validators_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/config/validators"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/go-http-utils/headers"
	"github.com/gobuffalo/validate"
	"github.com/stretchr/testify/assert"
)

const (
	testAPIPath        = "/api/test"
	testScriptContent  = "response.status = 200"
	testScriptFilePath = "/scripts/test.lua"
	scriptPathField    = "script.path"
	scriptScriptField  = "script.script"
	scriptFileField    = "script.file"
)

func TestScriptValidator(t *testing.T) {
	t.Run("should not register errors for valid inline script", func(t *testing.T) {
		errors := validate.Validate(&validators.ScriptValidator{
			Field: "script",
			Value: config.Script{
				Matcher: config.RequestMatcher{
					Path:   testAPIPath,
					Method: "GET",
				},
				Script: testScriptContent,
			},
		})

		assert.False(t, errors.HasAny())
	})

	t.Run("should not register errors for valid file script", func(t *testing.T) {
		fs := testutils.FsFromMap(t, map[string]string{
			testScriptFilePath: testScriptContent,
		})

		errors := validate.Validate(&validators.ScriptValidator{
			Field: "script",
			Value: config.Script{
				Matcher: config.RequestMatcher{
					Path:   testAPIPath,
					Method: "POST",
				},
				File: testScriptFilePath,
			},
			Fs: fs,
		})

		assert.False(t, errors.HasAny())
	})

	t.Run("should not register errors when method is empty", func(t *testing.T) {
		errors := validate.Validate(&validators.ScriptValidator{
			Field: "script",
			Value: config.Script{
				Matcher: config.RequestMatcher{
					Path:   testAPIPath,
					Method: "",
				},
				Script: testScriptContent,
			},
		})

		assert.False(t, errors.HasAny())
	})

	t.Run("should not register errors for valid queries and headers", func(t *testing.T) {
		errors := validate.Validate(&validators.ScriptValidator{
			Field: "script",
			Value: config.Script{
				Matcher: config.RequestMatcher{
					Path: "/api/test",
					Queries: map[string]string{
						"filter": "active",
						"sort":   "name",
					},
					Headers: map[string]string{
						"X-Custom-Header":     "value",
						headers.Authorization: "Bearer token",
					},
				},
				Script: testScriptContent,
			},
		})

		assert.False(t, errors.HasAny())
	})

	t.Run("should register error when path is empty", func(t *testing.T) {
		errors := validate.Validate(&validators.ScriptValidator{
			Field: "script",
			Value: config.Script{
				Matcher: config.RequestMatcher{
					Path: "",
				},
				Script: testScriptContent,
			},
		})

		assert.True(t, errors.HasAny())
		assert.Contains(t, errors.Error(), scriptPathField)
	})

	t.Run("should register error when path is invalid", func(t *testing.T) {
		errors := validate.Validate(&validators.ScriptValidator{
			Field: "script",
			Value: config.Script{
				Matcher: config.RequestMatcher{
					Path: "invalid-path",
				},
				Script: testScriptContent,
			},
		})

		assert.True(t, errors.HasAny())
		assert.Contains(t, errors.Error(), scriptPathField)
	})

	t.Run("should register error when method is invalid", func(t *testing.T) {
		errors := validate.Validate(&validators.ScriptValidator{
			Field: "script",
			Value: config.Script{
				Matcher: config.RequestMatcher{
					Path:   testAPIPath,
					Method: "INVALID",
				},
				Script: testScriptContent,
			},
		})

		assert.True(t, errors.HasAny())
		assert.Contains(t, errors.Error(), "script.method")
	})

	t.Run("should register error when neither script nor file is provided", func(t *testing.T) {
		errors := validate.Validate(&validators.ScriptValidator{
			Field: "script",
			Value: config.Script{
				Matcher: config.RequestMatcher{
					Path:   testAPIPath,
					Method: "GET",
				},
			},
		})

		assert.True(t, errors.HasAny())
		assert.Contains(t, errors.Error(), scriptScriptField)
		assert.Contains(t, errors.Error(), scriptFileField)
		assert.Contains(t, errors.Error(), "either 'script' or 'file' must be provided")
	})

	t.Run("should register error when both script and file are provided", func(t *testing.T) {
		fs := testutils.FsFromMap(t, map[string]string{
			testScriptFilePath: testScriptContent,
		})

		errors := validate.Validate(&validators.ScriptValidator{
			Field: "script",
			Value: config.Script{
				Matcher: config.RequestMatcher{
					Path: "/api/test",
				},
				Script: testScriptContent,
				File:   "/scripts/test.lua",
			},
			Fs: fs,
		})

		assert.True(t, errors.HasAny())
		assert.Contains(t, errors.Error(), scriptScriptField)
		assert.Contains(t, errors.Error(), scriptFileField)
		assert.Contains(t, errors.Error(), "only one of 'script' or 'file' can be provided")
	})

	t.Run("should register error when file does not exist", func(t *testing.T) {
		fs := testutils.FsFromMap(t, map[string]string{})

		errors := validate.Validate(&validators.ScriptValidator{
			Field: "script",
			Value: config.Script{
				Matcher: config.RequestMatcher{
					Path: "/api/test",
				},
				File: "/scripts/nonexistent.lua",
			},
			Fs: fs,
		})

		assert.True(t, errors.HasAny())
		assert.Contains(t, errors.Error(), scriptFileField)
	})

	t.Run("should register error when file is a directory", func(t *testing.T) {
		fs := testutils.FsFromMap(t, map[string]string{
			testScriptFilePath: testScriptContent,
		})

		errors := validate.Validate(&validators.ScriptValidator{
			Field: "script",
			Value: config.Script{
				Matcher: config.RequestMatcher{
					Path: "/api/test",
				},
				File: "/scripts",
			},
			Fs: fs,
		})

		assert.True(t, errors.HasAny())
		assert.Contains(t, errors.Error(), scriptFileField)
	})

	t.Run("should register multiple errors for complex invalid config", func(t *testing.T) {
		errors := validate.Validate(&validators.ScriptValidator{
			Field: "script",
			Value: config.Script{
				Matcher: config.RequestMatcher{
					Path:   "",
					Method: "INVALID",
				},
			},
		})

		assert.True(t, errors.HasAny())
		errString := errors.Error()
		assert.Contains(t, errString, "script.path")
		assert.Contains(t, errString, "script.method")
		assert.Contains(t, errString, "script.script")
		assert.Contains(t, errString, "script.file")
	})
}
