package config_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/go-http-utils/headers"
	"github.com/stretchr/testify/assert"
)

func TestRequestMatcherIsPathOnly(t *testing.T) {
	t.Run("returns true when only path is set", func(t *testing.T) {
		m := &config.RequestMatcher{Path: "/api/resource"}
		assert.True(t, m.IsPathOnly())
	})

	t.Run("returns false when method is set", func(t *testing.T) {
		m := &config.RequestMatcher{Path: "/api", Method: "GET"}
		assert.False(t, m.IsPathOnly())
	})

	t.Run("returns false when queries are set", func(t *testing.T) {
		m := &config.RequestMatcher{
			Path:    "/api",
			Queries: map[string]string{"key": "value"},
		}
		assert.False(t, m.IsPathOnly())
	})

	t.Run("returns false when headers are set", func(t *testing.T) {
		m := &config.RequestMatcher{
			Path:    "/api",
			Headers: map[string]string{"X-Token": "abc"},
		}
		assert.False(t, m.IsPathOnly())
	})

	t.Run("returns true when nothing is set", func(t *testing.T) {
		m := &config.RequestMatcher{}
		assert.True(t, m.IsPathOnly())
	})
}

const requestMatcherTestPath = "/api/test"

func TestRequestMatcherValidator(t *testing.T) {
	t.Run("should not register errors for valid filter with all fields", func(t *testing.T) {
		var errs config.Errors
		config.RequestMatcher{
			Path:   requestMatcherTestPath,
			Method: "GET",
			Queries: map[string]string{
				"param1": "value1",
				"param2": "value2",
			},
			Headers: map[string]string{
				headers.ContentType: "application/json",
				headers.Accept:      "application/json",
			},
		}.Validate("test", &errs)
		assert.False(t, errs.HasAny())
	})

	t.Run("should not register errors for valid filter with minimal fields", func(t *testing.T) {
		var errs config.Errors
		config.RequestMatcher{Path: requestMatcherTestPath}.Validate("test", &errs)
		assert.False(t, errs.HasAny())
	})

	t.Run("should register error for invalid path", func(t *testing.T) {
		var errs config.Errors
		config.RequestMatcher{Path: "", Method: "GET"}.Validate("test", &errs)
		assert.True(t, errs.HasAny())
		assert.Contains(t, errs.Error(), "path must not be empty")
	})

	t.Run("should register error for invalid method", func(t *testing.T) {
		var errs config.Errors
		config.RequestMatcher{Path: requestMatcherTestPath, Method: "INVALID"}.Validate("test", &errs)
		assert.True(t, errs.HasAny())
		assert.Contains(t, errs.Error(), "method must be one of")
	})

	t.Run("should register multiple validation errors", func(t *testing.T) {
		var errs config.Errors
		config.RequestMatcher{Path: "", Method: "INVALID"}.Validate("test", &errs)
		assert.True(t, errs.HasAny())
		assert.Contains(t, errs.Error(), "path must not be empty")
		assert.Contains(t, errs.Error(), "method must be one of")
	})
}
