package validators_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/config/validators"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/gobuffalo/validate"
	"github.com/stretchr/testify/assert"
)

func TestScriptValidator(t *testing.T) {
	t.Run("should not register errors for valid inline script", func(t *testing.T) {
		errors := validate.Validate(&validators.ScriptValidator{
			Field: "script",
			Value: config.Script{
				RequestMatcher: config.RequestMatcher{
					Path:   "/api/test",
					Method: "GET",
				},
				Script: "response.status = 200",
			},
		})

		assert.False(t, errors.HasAny())
	})

	t.Run("should not register errors for valid file script", func(t *testing.T) {
		fs := testutils.FsFromMap(t, map[string]string{
			"/scripts/test.lua": "response.status = 200",
		})

		errors := validate.Validate(&validators.ScriptValidator{
			Field: "script",
			Value: config.Script{
				RequestMatcher: config.RequestMatcher{
					Path:   "/api/test",
					Method: "POST",
				},
				File: "/scripts/test.lua",
			},
			Fs: fs,
		})

		assert.False(t, errors.HasAny())
	})

	t.Run("should not register errors when method is empty", func(t *testing.T) {
		errors := validate.Validate(&validators.ScriptValidator{
			Field: "script",
			Value: config.Script{
				RequestMatcher: config.RequestMatcher{
					Path:   "/api/test",
					Method: "",
				},
				Script: "response.status = 200",
			},
		})

		assert.False(t, errors.HasAny())
	})

	t.Run("should not register errors for valid queries and headers", func(t *testing.T) {
		errors := validate.Validate(&validators.ScriptValidator{
			Field: "script",
			Value: config.Script{
				RequestMatcher: config.RequestMatcher{
					Path: "/api/test",
					Queries: map[string]string{
						"filter": "active",
						"sort":   "name",
					},
					Headers: map[string]string{
						"X-Custom-Header": "value",
						"Authorization":   "Bearer token",
					},
				},
				Script: "response.status = 200",
			},
		})

		assert.False(t, errors.HasAny())
	})

	t.Run("should register error when path is empty", func(t *testing.T) {
		errors := validate.Validate(&validators.ScriptValidator{
			Field: "script",
			Value: config.Script{
				RequestMatcher: config.RequestMatcher{
					Path: "",
				},
				Script: "response.status = 200",
			},
		})

		assert.True(t, errors.HasAny())
		assert.Contains(t, errors.Error(), "script.path")
	})

	t.Run("should register error when path is invalid", func(t *testing.T) {
		errors := validate.Validate(&validators.ScriptValidator{
			Field: "script",
			Value: config.Script{
				RequestMatcher: config.RequestMatcher{
					Path: "invalid-path",
				},
				Script: "response.status = 200",
			},
		})

		assert.True(t, errors.HasAny())
		assert.Contains(t, errors.Error(), "script.path")
	})

	t.Run("should register error when method is invalid", func(t *testing.T) {
		errors := validate.Validate(&validators.ScriptValidator{
			Field: "script",
			Value: config.Script{
				RequestMatcher: config.RequestMatcher{
					Path:   "/api/test",
					Method: "INVALID",
				},
				Script: "response.status = 200",
			},
		})

		assert.True(t, errors.HasAny())
		assert.Contains(t, errors.Error(), "script.method")
	})

	t.Run("should register error when neither script nor file is provided", func(t *testing.T) {
		errors := validate.Validate(&validators.ScriptValidator{
			Field: "script",
			Value: config.Script{
				RequestMatcher: config.RequestMatcher{
					Path:   "/api/test",
					Method: "GET",
				},
			},
		})

		assert.True(t, errors.HasAny())
		assert.Contains(t, errors.Error(), "script.script")
		assert.Contains(t, errors.Error(), "script.file")
		assert.Contains(t, errors.Error(), "either 'script' or 'file' must be provided")
	})

	t.Run("should register error when both script and file are provided", func(t *testing.T) {
		fs := testutils.FsFromMap(t, map[string]string{
			"/scripts/test.lua": "response.status = 200",
		})

		errors := validate.Validate(&validators.ScriptValidator{
			Field: "script",
			Value: config.Script{
				RequestMatcher: config.RequestMatcher{
					Path: "/api/test",
				},
				Script: "response.status = 200",
				File:   "/scripts/test.lua",
			},
			Fs: fs,
		})

		assert.True(t, errors.HasAny())
		assert.Contains(t, errors.Error(), "script.script")
		assert.Contains(t, errors.Error(), "script.file")
		assert.Contains(t, errors.Error(), "only one of 'script' or 'file' can be provided")
	})

	t.Run("should register error when file does not exist", func(t *testing.T) {
		fs := testutils.FsFromMap(t, map[string]string{})

		errors := validate.Validate(&validators.ScriptValidator{
			Field: "script",
			Value: config.Script{
				RequestMatcher: config.RequestMatcher{
					Path: "/api/test",
				},
				File: "/scripts/nonexistent.lua",
			},
			Fs: fs,
		})

		assert.True(t, errors.HasAny())
		assert.Contains(t, errors.Error(), "script.file")
	})

	t.Run("should register error when file is a directory", func(t *testing.T) {
		fs := testutils.FsFromMap(t, map[string]string{
			"/scripts/test.lua": "response.status = 200",
		})

		errors := validate.Validate(&validators.ScriptValidator{
			Field: "script",
			Value: config.Script{
				RequestMatcher: config.RequestMatcher{
					Path: "/api/test",
				},
				File: "/scripts",
			},
			Fs: fs,
		})

		assert.True(t, errors.HasAny())
		assert.Contains(t, errors.Error(), "script.file")
	})

	t.Run("should register error when query key is empty", func(t *testing.T) {
		errors := validate.Validate(&validators.ScriptValidator{
			Field: "script",
			Value: config.Script{
				RequestMatcher: config.RequestMatcher{
					Path: "/api/test",
					Queries: map[string]string{
						"":     "value",
						"sort": "name",
					},
				},
				Script: "response.status = 200",
			},
		})

		assert.True(t, errors.HasAny())
		assert.Contains(t, errors.Error(), "script.queries")
		assert.Contains(t, errors.Error(), "query parameter keys must not be empty")
	})

	t.Run("should register error when header key is empty", func(t *testing.T) {
		errors := validate.Validate(&validators.ScriptValidator{
			Field: "script",
			Value: config.Script{
				RequestMatcher: config.RequestMatcher{
					Path: "/api/test",
					Headers: map[string]string{
						"":            "value",
						"X-Custom-ID": "123",
					},
				},
				Script: "response.status = 200",
			},
		})

		assert.True(t, errors.HasAny())
		assert.Contains(t, errors.Error(), "script.headers")
		assert.Contains(t, errors.Error(), "header keys must not be empty")
	})

	t.Run("should register multiple errors for complex invalid config", func(t *testing.T) {
		errors := validate.Validate(&validators.ScriptValidator{
			Field: "script",
			Value: config.Script{
				RequestMatcher: config.RequestMatcher{
					Path:   "",
					Method: "INVALID",
					Queries: map[string]string{
						"": "empty-key",
					},
					Headers: map[string]string{
						"": "empty-header-key",
					},
				},
			},
		})

		assert.True(t, errors.HasAny())
		errString := errors.Error()
		assert.Contains(t, errString, "script.path")
		assert.Contains(t, errString, "script.method")
		assert.Contains(t, errString, "script.script")
		assert.Contains(t, errString, "script.file")
		assert.Contains(t, errString, "script.queries")
		assert.Contains(t, errString, "script.headers")
	})
}
