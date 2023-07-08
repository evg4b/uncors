package helpers_test

import (
	"net/http"
	"testing"

	"github.com/evg4b/uncors/internal/helpers"
	"github.com/stretchr/testify/assert"
)

func TestAssertIsDefined(t *testing.T) {
	t.Run("where value is nil", func(t *testing.T) {
		t.Run("should panic on", func(t *testing.T) {
			t.Run("just nil value", func(t *testing.T) {
				assert.Panics(t, func() {
					helpers.AssertIsDefined(nil)
				})
			})
			t.Run("nil value in pointer", func(t *testing.T) {
				assert.Panics(t, func() {
					helpers.AssertIsDefined((*int64)(nil))
				})
			})
			t.Run("nil value in interface", func(t *testing.T) {
				assert.Panics(t, func() {
					helpers.AssertIsDefined((http.Handler)(nil))
				})
			})
			t.Run("nil value in function", func(t *testing.T) {
				assert.Panics(t, func() {
					var f func()
					helpers.AssertIsDefined(f)
				})
			})
		})

		t.Run("should panic with ", func(t *testing.T) {
			tests := []struct {
				name     string
				message  []string
				expected string
			}{
				{
					name:     "default message where message is not set",
					message:  []string{},
					expected: "Required variable is not defined",
				},
				{
					name:     "custom message where it is set",
					message:  []string{"Custom message"},
					expected: "Custom message",
				},
				{
					name:     "concatenation of all passed messages",
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
	})
}
