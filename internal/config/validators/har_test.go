package validators_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/config/validators"
	"github.com/gobuffalo/validate"
	"github.com/stretchr/testify/assert"
)

func TestHARValidator(t *testing.T) {
	t.Run("valid cases", func(t *testing.T) {
		cases := []struct {
			name  string
			value config.HARConfig
		}{
			{
				name:  "disabled (empty file)",
				value: config.HARConfig{},
			},
			{
				name:  "valid file path with extension",
				value: config.HARConfig{File: "output.har"},
			},
			{
				name:  "path with directory and extension",
				value: config.HARConfig{File: "/tmp/trace.har"},
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				errs := validate.Validate(&validators.HARValidator{
					Field: "mappings[0].har",
					Value: tc.value,
				})

				assert.False(t, errs.HasAny())
			})
		}
	})

	t.Run("invalid cases", func(t *testing.T) {
		t.Run("file path without extension", func(t *testing.T) {
			errs := validate.Validate(&validators.HARValidator{
				Field: "mappings[0].har",
				Value: config.HARConfig{File: "outputfile"},
			})

			assert.True(t, errs.HasAny())
		})
	})
}
