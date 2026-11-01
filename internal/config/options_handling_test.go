package config_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/headers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptionsValidator(t *testing.T) {
	t.Run("should return true", func(t *testing.T) {
		t.Run("for default options", func(t *testing.T) {
			assert.NoError(t, (&config.OptionsHandling{}).Validate("options"))
		})

		t.Run("for correct status code", func(t *testing.T) {
			assert.NoError(t, (&config.OptionsHandling{
				Headers: map[string]string{headers.ContentType: "application/json"},
				Code:    200,
			}).Validate("options"))
		})
	})

	t.Run("should return false for invalid status code", func(t *testing.T) {
		require.EqualError(t, (&config.OptionsHandling{
			Headers: map[string]string{headers.ContentType: "application/json"},
			Code:    -10,
		}).Validate("options"), "options.code must be in range 200-599")
	})
}
