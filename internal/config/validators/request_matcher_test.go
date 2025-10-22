package validators_test

import (
	"testing"

	"github.com/gobuffalo/validate"
	"github.com/stretchr/testify/assert"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/config/validators"
)

func TestRequestMatcherValidator(t *testing.T) {
	t.Run("should not register errors for valid filter with all fields", func(t *testing.T) {
		errors := validate.Validate(&validators.RequestMatcherValidator{
			Field: "test",
			Value: config.RequestMatcher{
				Path:   "/api/test",
				Method: "GET",
				Queries: map[string]string{
					"param1": "value1",
					"param2": "value2",
				},
				Headers: map[string]string{
					"Content-Type": "application/json",
					"Accept":       "application/json",
				},
			},
		})

		assert.False(t, errors.HasAny())
	})

	t.Run("should not register errors for valid filter with minimal fields", func(t *testing.T) {
		errors := validate.Validate(&validators.RequestMatcherValidator{
			Field: "test",
			Value: config.RequestMatcher{
				Path: "/api/test",
			},
		})

		assert.False(t, errors.HasAny())
	})

	t.Run("should register error for invalid path", func(t *testing.T) {
		errors := validate.Validate(&validators.RequestMatcherValidator{
			Field: "test",
			Value: config.RequestMatcher{
				Path:   "",
				Method: "GET",
			},
		})

		assert.True(t, errors.HasAny())
		assert.Contains(t, errors.Error(), "path must not be empty")
	})

	t.Run("should register error for invalid method", func(t *testing.T) {
		errors := validate.Validate(&validators.RequestMatcherValidator{
			Field: "test",
			Value: config.RequestMatcher{
				Path:   "/api/test",
				Method: "INVALID",
			},
		})

		assert.True(t, errors.HasAny())
		assert.Contains(t, errors.Error(), "method must be one of")
	})

	t.Run("should register multiple validation errors", func(t *testing.T) {
		errors := validate.Validate(&validators.RequestMatcherValidator{
			Field: "test",
			Value: config.RequestMatcher{
				Path:   "",
				Method: "INVALID",
			},
		})

		assert.True(t, errors.HasAny())
		errMsg := errors.Error()
		assert.Contains(t, errMsg, "path must not be empty")
		assert.Contains(t, errMsg, "method must be one of")
	})
}
