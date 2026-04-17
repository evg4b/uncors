package validators_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/config/validators"
	"github.com/go-http-utils/headers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptionsValidator(t *testing.T) {
	t.Run("should return true", func(t *testing.T) {
		t.Run("for default options", func(t *testing.T) {
			var errs validators.Errors
			validators.ValidateOptionsHandling("options", config.OptionsHandling{}, &errs)
			assert.False(t, errs.HasAny())
		})

		t.Run("for correct status code", func(t *testing.T) {
			var errs validators.Errors
			validators.ValidateOptionsHandling("options", config.OptionsHandling{
				Headers: map[string]string{headers.ContentType: "application/json"},
				Code:    200,
			}, &errs)
			assert.False(t, errs.HasAny())
		})
	})

	t.Run("should return false for invalid status code", func(t *testing.T) {
		var errs validators.Errors
		validators.ValidateOptionsHandling("options", config.OptionsHandling{
			Headers: map[string]string{headers.ContentType: "application/json"},
			Code:    -10,
		}, &errs)
		require.EqualError(t, errs, "options.code code must be in range 100-599")
	})
}
