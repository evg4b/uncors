package validators_test

import (
	"fmt"
	"testing"

	"github.com/evg4b/uncors/internal/config/validators"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheValidator(t *testing.T) {
	const field = "cache"

	t.Run("should not register errors for", func(t *testing.T) {
		patterns := []string{"/api/**", "/constants", "/translations", "/**/*.js", "/**", "/[12]/demo", "**", "*"}
		for _, pattern := range patterns {
			p := pattern
			t.Run(fmt.Sprintf("%s pattern", p), func(t *testing.T) {
				var errs validators.Errors
				validators.ValidateCacheGlob(field, p, &errs)
				assert.False(t, errs.HasAny())
			})
		}
	})

	t.Run("should register errors for", func(t *testing.T) {
		tests := []struct{ pattern, error string }{
			{pattern: "/[12/demo", error: "cache is not a valid glob pattern"},
			{pattern: "/{{12}/demo", error: "cache is not a valid glob pattern"},
		}
		for _, test := range tests {
			t.Run(fmt.Sprintf("%s test", test.pattern), func(t *testing.T) {
				var errs validators.Errors
				validators.ValidateCacheGlob(field, test.pattern, &errs)
				require.EqualError(t, errs, test.error)
			})
		}
	})
}
