package base_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config/validators/base"
	"github.com/gobuffalo/validate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const option1 = "option-1"
const option2 = "option-2"

func TestStringEnumValidator(t *testing.T) {
	t.Run("valid option", func(t *testing.T) {
		errors := validate.Validate(&base.StringEnumValidator{
			Field: "field",
			Value: option1,
			Options: []string{
				option1,
				option2,
			},
		})

		assert.False(t, errors.HasAny())
	})

	t.Run("valid option", func(t *testing.T) {
		errors := validate.Validate(&base.StringEnumValidator{
			Field: "field",
			Value: "option-x",
			Options: []string{
				option1,
				option2,
			},
		})

		require.EqualError(t, errors, "'option-x' is not a valid option")
	})
}
