package helpers_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/helpers"
	"github.com/stretchr/testify/assert"
)

func TestAssertIsDefined(t *testing.T) {
	t.Run("where value is nil", func(t *testing.T) {
		tests := []struct {
			name     string
			message  []string
			expected string
		}{
			{
				name:     "shiuld panic with default message where message is not set",
				message:  []string{},
				expected: "Requared variable is not defined",
			},
			{
				name:     "shiuld panic with custom message where it is set",
				message:  []string{"Cusom message"},
				expected: "Cusom message",
			},
			{
				name:     "shiuld panic with concatanetion of all passed messages",
				message:  []string{"This", "is", "custom", "message"},
				expected: "This is custom message",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				assert.PanicsWithValue(t, tt.expected, func() {
					helpers.AssertIsDefined(nil, tt.message...)
				})
			})
		}
	})
}
