package validators_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/config/validators"
	"github.com/stretchr/testify/assert"
)

func TestValidateHostsFileEntries(t *testing.T) {
	t.Run("should not panic with valid config", func(t *testing.T) {
		cfg := &config.UncorsConfig{
			Mappings: []config.Mapping{
				{From: "http://localhost:8080", To: "https://example.com"},
			},
		}

		assert.NotPanics(t, func() {
			validators.ValidateHostsFileEntries(cfg)
		})
	})

	t.Run("should handle wildcard mappings", func(t *testing.T) {
		cfg := &config.UncorsConfig{
			Mappings: []config.Mapping{
				{From: "http://*.local:8080", To: "https://*.example.com"},
			},
		}

		assert.NotPanics(t, func() {
			validators.ValidateHostsFileEntries(cfg)
		})
	})

	t.Run("should handle empty mappings", func(t *testing.T) {
		cfg := &config.UncorsConfig{
			Mappings: []config.Mapping{},
		}

		assert.NotPanics(t, func() {
			validators.ValidateHostsFileEntries(cfg)
		})
	})

	t.Run("should handle invalid URLs gracefully", func(t *testing.T) {
		cfg := &config.UncorsConfig{
			Mappings: []config.Mapping{
				{From: "not-a-url", To: "https://example.com"},
			},
		}

		assert.NotPanics(t, func() {
			validators.ValidateHostsFileEntries(cfg)
		})
	})
}
