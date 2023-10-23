package validators_test

import (
	"fmt"
	"github.com/evg4b/uncors/internal/config/validators"
	v "github.com/gobuffalo/validate"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestIsValid(t *testing.T) {
	const field = "PortField"

	t.Run("valid port", func(t *testing.T) {
		for _, port := range []int{1, 443, 65535} {
			t.Run(fmt.Sprintf("port %d", port), func(t *testing.T) {
				validator := &validators.PortValidator{Field: field, Value: port}
				errors := v.NewErrors()
				validator.IsValid(errors)

				assert.False(t, errors.HasAny())
			})
		}
	})

	t.Run("invalid port", func(t *testing.T) {
		for _, port := range []int{-5, 0, 70000} {
			t.Run(fmt.Sprintf("port %d", port), func(t *testing.T) {
				validator := &validators.PortValidator{Field: field, Value: port}
				errors := v.NewErrors()
				validator.IsValid(errors)

				assert.True(t, errors.HasAny())

				assert.Equal(t, []string{"PortField must be between 0 and 65535"}, errors.Get(field))
			})
		}
	})
}
