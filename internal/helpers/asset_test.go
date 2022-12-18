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
				name:     "should panic with default message where message is not set",
				message:  []string{},
				expected: "Required variable is not defined",
			},
			{
				name:     "should panic with custom message where it is set",
				message:  []string{"Custom message"},
				expected: "Custom message",
			},
			{
				name:     "should panic with concatenation of all passed messages",
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
