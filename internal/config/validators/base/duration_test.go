package base_test

import (
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config/validators/base"

	"github.com/gobuffalo/validate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDurationValidator(t *testing.T) {
	const field = "test-field"

	t.Run("should not register errors for", func(t *testing.T) {
		tests := []struct {
			name      string
			value     time.Duration
			allowZero bool
		}{
			{
				name:  "positive value without allow zero",
				value: 1 * time.Second,
			},
			{
				name:      "zero value with allow zero",
				value:     0,
				allowZero: true,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				errors := validate.Validate(&base.DurationValidator{
					Field:     field,
					Value:     tt.value,
					AllowZero: tt.allowZero,
				})

				assert.False(t, errors.HasAny())
			})
		}
	})

	t.Run("should register errors for", func(t *testing.T) {
		tests := []struct {
			name      string
			value     time.Duration
			allowZero bool
			error     string
		}{
			{
				name:  "negative value without allow zero",
				value: -1 * time.Second,
				error: "test-field must be greater than 0",
			},
			{
				name:  "zero value without allow zero",
				value: 0,
				error: "test-field must be greater than 0",
			},
			{
				name:      "negative value with allow zero",
				value:     -1 * time.Second,
				error:     "test-field must be greater than or equal to 0",
				allowZero: true,
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				errors := validate.Validate(&base.DurationValidator{
					Field:     field,
					Value:     test.value,
					AllowZero: test.allowZero,
				})

				require.EqualError(t, errors, test.error)
			})
		}
	})
}
