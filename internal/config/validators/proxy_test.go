package validators_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config/validators"

	"github.com/gobuffalo/validate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProxyValidatorIsValid(t *testing.T) {
	t.Run("valid url", func(t *testing.T) {
		err := validate.Validate(&validators.ProxyValidator{
			Field: "testField",
			Value: "http://valid-url.com",
		})

		assert.False(t, err.HasAny())
	})

	t.Run("invalid url", func(t *testing.T) {
		err := validate.Validate(&validators.ProxyValidator{
			Field: "testField",
			Value: "invalid:::url",
		})

		assert.NotEmpty(t, err)
		require.EqualError(t, err, "testField is not a valid URL")
	})

	t.Run("empty url", func(t *testing.T) {
		err := validate.Validate(&validators.ProxyValidator{
			Field: "testField",
			Value: "",
		})

		assert.False(t, err.HasAny())
	})
}
