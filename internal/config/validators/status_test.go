package validators_test

import (
	"strconv"
	"testing"

	"github.com/evg4b/uncors/internal/config/validators"

	"github.com/gobuffalo/validate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatusValidator(t *testing.T) {
	const field = "status"
	t.Run("valid status code", func(t *testing.T) {
		validCodes := []int{
			100,
			200,
			300,
			400,
			404,
			500,
			503,
			599,
		}

		for _, code := range validCodes {
			t.Run(strconv.Itoa(code), func(t *testing.T) {
				errors := validate.Validate(&validators.StatusValidator{
					Field: field,
					Value: code,
				})

				assert.False(t, errors.HasAny())
			})
		}
	})

	t.Run("invalid status code", func(t *testing.T) {
		invalidCodes := []int{
			-200,
			0,
			99,
			600,
		}

		for _, code := range invalidCodes {
			t.Run(strconv.Itoa(code), func(t *testing.T) {
				errors := validate.Validate(&validators.StatusValidator{
					Field: field,
					Value: code,
				})

				require.EqualError(t, errors, "status code must be in range 100-599")
			})
		}
	})
}
