package validators_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/config/validators"
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
				errs := &validators.Errors{}
				validators.ValidateHAR("mappings[0].har", tc.value, errs)

				assert.False(t, errs.HasAny())
			})
		}
	})

	t.Run("invalid cases", func(t *testing.T) {
		t.Run("file path without extension", func(t *testing.T) {
			errs := &validators.Errors{}
			validators.ValidateHAR("mappings[0].har", config.HARConfig{File: "outputfile"}, errs)

			assert.True(t, errs.HasAny())
		})
	})
}
