package validators_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/config/validators"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/go-http-utils/headers"
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
	noFS := testutils.FsFromMap(t, map[string]string{})

	t.Run("valid inline script", func(t *testing.T) {
		var errs validators.Errors
		validators.ValidateScript("script", config.Script{
			Matcher: config.RequestMatcher{Path: testAPIPath, Method: "GET"},
			Script:  testScriptContent,
		}, noFS, &errs)
		assert.False(t, errs.HasAny())
	})

	t.Run("valid file script", func(t *testing.T) {
		fs := testutils.FsFromMap(t, map[string]string{testScriptFilePath: testScriptContent})

		var errs validators.Errors
		validators.ValidateScript("script", config.Script{
			Matcher: config.RequestMatcher{Path: testAPIPath, Method: "POST"},
			File:    testScriptFilePath,
		}, fs, &errs)
		assert.False(t, errs.HasAny())
	})

	t.Run("empty method is allowed", func(t *testing.T) {
		var errs validators.Errors
		validators.ValidateScript("script", config.Script{
			Matcher: config.RequestMatcher{Path: testAPIPath},
			Script:  testScriptContent,
		}, noFS, &errs)
		assert.False(t, errs.HasAny())
	})

	t.Run("valid queries and headers", func(t *testing.T) {
		var errs validators.Errors
		validators.ValidateScript("script", config.Script{
			Matcher: config.RequestMatcher{
				Path:    "/api/test",
				Queries: map[string]string{"filter": "active"},
				Headers: map[string]string{headers.Authorization: "Bearer token"},
			},
			Script: testScriptContent,
		}, noFS, &errs)
		assert.False(t, errs.HasAny())
	})

	t.Run("empty path", func(t *testing.T) {
		var errs validators.Errors
		validators.ValidateScript("script", config.Script{
			Matcher: config.RequestMatcher{Path: ""},
			Script:  testScriptContent,
		}, noFS, &errs)
		assert.True(t, errs.HasAny())
		assert.Contains(t, errs.Error(), scriptPathField)
	})

	t.Run("invalid path", func(t *testing.T) {
		var errs validators.Errors
		validators.ValidateScript("script", config.Script{
			Matcher: config.RequestMatcher{Path: "invalid-path"},
			Script:  testScriptContent,
		}, noFS, &errs)
		assert.True(t, errs.HasAny())
		assert.Contains(t, errs.Error(), scriptPathField)
	})

	t.Run("invalid method", func(t *testing.T) {
		var errs validators.Errors
		validators.ValidateScript("script", config.Script{
			Matcher: config.RequestMatcher{Path: testAPIPath, Method: "INVALID"},
			Script:  testScriptContent,
		}, noFS, &errs)
		assert.True(t, errs.HasAny())
		assert.Contains(t, errs.Error(), "script.method")
	})

	t.Run("neither script nor file provided", func(t *testing.T) {
		var errs validators.Errors
		validators.ValidateScript("script", config.Script{
			Matcher: config.RequestMatcher{Path: testAPIPath, Method: "GET"},
		}, noFS, &errs)
		assert.True(t, errs.HasAny())
		assert.Contains(t, errs.Error(), scriptScriptField)
		assert.Contains(t, errs.Error(), scriptFileField)
		assert.Contains(t, errs.Error(), "either 'script' or 'file' must be provided")
	})

	t.Run("both script and file provided", func(t *testing.T) {
		fs := testutils.FsFromMap(t, map[string]string{testScriptFilePath: testScriptContent})

		var errs validators.Errors
		validators.ValidateScript("script", config.Script{
			Matcher: config.RequestMatcher{Path: "/api/test"},
			Script:  testScriptContent,
			File:    "/scripts/test.lua",
		}, fs, &errs)
		assert.True(t, errs.HasAny())
		assert.Contains(t, errs.Error(), scriptScriptField)
		assert.Contains(t, errs.Error(), scriptFileField)
		assert.Contains(t, errs.Error(), "only one of 'script' or 'file' can be provided")
	})

	t.Run("file does not exist", func(t *testing.T) {
		var errs validators.Errors
		validators.ValidateScript("script", config.Script{
			Matcher: config.RequestMatcher{Path: "/api/test"},
			File:    "/scripts/nonexistent.lua",
		}, noFS, &errs)
		assert.True(t, errs.HasAny())
		assert.Contains(t, errs.Error(), scriptFileField)
	})

	t.Run("multiple errors", func(t *testing.T) {
		var errs validators.Errors
		validators.ValidateScript("script", config.Script{
			Matcher: config.RequestMatcher{Path: "", Method: "INVALID"},
		}, noFS, &errs)
		assert.True(t, errs.HasAny())
		errStr := errs.Error()
		assert.Contains(t, errStr, "script.path")
		assert.Contains(t, errStr, "script.method")
		assert.Contains(t, errStr, "script.script")
		assert.Contains(t, errStr, "script.file")
	})
}
