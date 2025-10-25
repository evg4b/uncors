package validators_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/config/validators"
	"github.com/go-http-utils/headers"
	"github.com/gobuffalo/validate"
	"github.com/stretchr/testify/assert"
)

func TestOptionsValidator(t *testing.T) {
	t.Run("should return true", func(t *testing.T) {
		t.Run("for default options", func(t *testing.T) {
			errors := validate.Validate(&validators.OptionsHandlingValidator{
				Field: "options",
				Value: config.OptionsHandling{},
			})

			assert.False(t, errors.HasAny())
		})

		t.Run("for correct status code", func(t *testing.T) {
			errors := validate.Validate(&validators.OptionsHandlingValidator{
				Field: "options",
				Value: config.OptionsHandling{
					Headers: map[string]string{
						headers.ContentType: "application/json",
					},
					Code: 200,
				},
			})

			assert.False(t, errors.HasAny())
		})
	})

	t.Run("should return false", func(t *testing.T) {
		errors := validate.Validate(&validators.OptionsHandlingValidator{
			Field: "options",
			Value: config.OptionsHandling{
				Headers: map[string]string{
					headers.ContentType: "application/json",
				},
				Code: -10,
			},
		})

		assert.True(t, errors.HasAny())
		assert.Equal(t, 1, errors.Count())
		assert.Equal(t, "options.code code must be in range 100-599", errors.Errors["options.code"][0])
	})
}
