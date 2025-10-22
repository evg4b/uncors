package validators_test

import (
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/config/validators"
	"github.com/gobuffalo/validate"
	"github.com/stretchr/testify/assert"
)

func TestMockValidator(t *testing.T) {
	t.Run("should return true", func(t *testing.T) {
		errors := validate.Validate(&validators.MockValidator{
			Field: "mock",
			Value: config.Mock{
				Matcher: config.RequestMatcher{
					Path:   "/api/info",
					Method: "",
				},
				Response: config.Response{
					Code:  200,
					Raw:   "test",
					Delay: 1 * time.Second,
				},
			},
		})

		assert.False(t, errors.HasAny())
	})
}
