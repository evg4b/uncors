package validators_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/evg4b/uncors/internal/config/validators"
	v "github.com/gobuffalo/validate"
	"github.com/stretchr/testify/assert"
)

func TestIsValid(t *testing.T) {
	const field = "port-field"

	t.Run("valid port", func(t *testing.T) {
		for _, port := range []int{1, 443, 65535} {
			t.Run(fmt.Sprintf("port %d", port), func(t *testing.T) {
				errors := v.Validate(&validators.PortValidator{
					Field: field,
					Value: port,
				})

				assert.False(t, errors.HasAny())
			})
		}
	})

	t.Run("invalid port", func(t *testing.T) {
		for _, port := range []int{-5, 0, 70000} {
			t.Run(fmt.Sprintf("port %d", port), func(t *testing.T) {
				errors := v.Validate(&validators.PortValidator{
					Field: field,
					Value: port,
				})

				require.EqualError(t, errors, "port-field must be between 0 and 65535")
			})
		}
	})
}
