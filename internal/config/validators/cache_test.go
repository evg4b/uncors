package validators_test

import (
	"fmt"
	"testing"

	"github.com/evg4b/uncors/internal/config/validators"
	"github.com/gobuffalo/validate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheValidator(t *testing.T) {
	const field = "cache"

	t.Run("should not register errors for", func(t *testing.T) {
		patterns := []string{
			"/api/**",
			"/constants",
			"/translations",
			"/**/*.js",
			"/**",
			"/[12]/demo",
			"**",
			"*",
		}
		for _, pattern := range patterns {
			t.Run(fmt.Sprintf("%s pattern", pattern), func(t *testing.T) {
				errors := validate.Validate(&validators.CacheValidator{
					Field: field,
					Value: pattern,
				})

				assert.False(t, errors.HasAny())
			})
		}
	})

	t.Run("should register errors for", func(t *testing.T) {
		tests := []struct {
			pattern string
			error   string
		}{
			{
				pattern: "/[12/demo",
				error:   "cache is not a valid glob pattern",
			},
			{
				pattern: "/{{12}/demo",
				error:   "cache is not a valid glob pattern",
			},
		}
		for _, test := range tests {
			t.Run(fmt.Sprintf("%s test", test), func(t *testing.T) {
				errors := validate.Validate(&validators.CacheValidator{
					Field: field,
					Value: test.pattern,
				})

				require.EqualError(t, errors, test.error)
			})
		}
	})
}
