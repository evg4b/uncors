package validators_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/config/validators"
	"github.com/go-http-utils/headers"
	"github.com/stretchr/testify/assert"
)

const requestMatcherTestPath = "/api/test"

func TestRequestMatcherValidator(t *testing.T) {
	t.Run("should not register errors for valid filter with all fields", func(t *testing.T) {
		var errs validators.Errors
		validators.ValidateRequestMatcher("test", config.RequestMatcher{
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
		}, &errs)
		assert.False(t, errs.HasAny())
	})

	t.Run("should not register errors for valid filter with minimal fields", func(t *testing.T) {
		var errs validators.Errors
		validators.ValidateRequestMatcher("test", config.RequestMatcher{Path: requestMatcherTestPath}, &errs)
		assert.False(t, errs.HasAny())
	})

	t.Run("should register error for invalid path", func(t *testing.T) {
		var errs validators.Errors
		validators.ValidateRequestMatcher("test", config.RequestMatcher{Path: "", Method: "GET"}, &errs)
		assert.True(t, errs.HasAny())
		assert.Contains(t, errs.Error(), "path must not be empty")
	})

	t.Run("should register error for invalid method", func(t *testing.T) {
		var errs validators.Errors
		validators.ValidateRequestMatcher(
			"test",
			config.RequestMatcher{Path: requestMatcherTestPath, Method: "INVALID"},
			&errs,
		)
		assert.True(t, errs.HasAny())
		assert.Contains(t, errs.Error(), "method must be one of")
	})

	t.Run("should register multiple validation errors", func(t *testing.T) {
		var errs validators.Errors
		validators.ValidateRequestMatcher("test", config.RequestMatcher{Path: "", Method: "INVALID"}, &errs)
		assert.True(t, errs.HasAny())
		assert.Contains(t, errs.Error(), "path must not be empty")
		assert.Contains(t, errs.Error(), "method must be one of")
	})
}
