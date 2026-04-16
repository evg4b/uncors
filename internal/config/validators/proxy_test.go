package validators_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config/validators"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProxyValidatorIsValid(t *testing.T) {
	t.Run("valid url", func(t *testing.T) {
		var errs validators.Errors
		validators.ValidateProxy("testField", "http://valid-url.com", &errs)
		assert.False(t, errs.HasAny())
	})

	t.Run("invalid url", func(t *testing.T) {
		var errs validators.Errors
		validators.ValidateProxy("testField", "invalid:::url", &errs)
		require.EqualError(t, errs, "testField is not a valid URL")
	})

	t.Run("empty url", func(t *testing.T) {
		var errs validators.Errors
		validators.ValidateProxy("testField", "", &errs)
		assert.False(t, errs.HasAny())
	})
}
