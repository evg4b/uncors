package base_test

import (
	"github.com/evg4b/uncors/internal/config/validators/base"
	"github.com/gobuffalo/validate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestStringEnumValidator(t *testing.T) {
	t.Run("valid option", func(t *testing.T) {
		errors := validate.Validate(&base.StringEnumValidator{
			Field: "field",
			Value: "option-1",
			Options: []string{
				"option-1",
				"option-2",
			},
		})

		assert.False(t, errors.HasAny())
	})

	t.Run("valid option", func(t *testing.T) {
		errors := validate.Validate(&base.StringEnumValidator{
			Field: "field",
			Value: "option-x",
			Options: []string{
				"option-1",
				"option-2",
			},
		})

		require.EqualError(t, errors, "'option-x' is not a valid option")
	})
}
