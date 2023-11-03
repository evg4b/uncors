package validators_test

import (
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config/validators"
	"github.com/gobuffalo/validate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDurationValidator(t *testing.T) {
	const field = "test-field"

	t.Run("should not register errors for", func(t *testing.T) {
		tests := []struct {
			name  string
			Value time.Duration
		}{
			{
				name:  "positive value",
				Value: 1 * time.Second,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				errors := validate.Validate(&validators.DurationValidator{
					Field: field,
					Value: tt.Value,
				})

				assert.False(t, errors.HasAny())
			})
		}
	})

	t.Run("should register errors for", func(t *testing.T) {
		tests := []struct {
			name  string
			Value time.Duration
			error string
		}{
			{
				name:  "zero value",
				Value: 0,
				error: "test-field must be greater than 0",
			},
			{
				name:  "negative value",
				Value: -1 * time.Second,
				error: "test-field must be greater than 0",
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				errors := validate.Validate(&validators.DurationValidator{
					Field: field,
					Value: test.Value,
				})

				require.EqualError(t, errors, test.error)
			})
		}
	})
}
