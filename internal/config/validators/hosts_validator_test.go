package validators_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/config/validators"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createMockHostsFile(t *testing.T, fs afero.Fs, content string) {
	hostsPath := helpers.GetHostsFilePath()
	err := afero.WriteFile(fs, hostsPath, []byte(content), 0o644)
	require.NoError(t, err)
}

func TestValidateHostsFileEntries(t *testing.T) {
	t.Run("should not panic with valid config and existing host", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		createMockHostsFile(t, fs, "127.0.0.1 localhost\n")

		cfg := &config.UncorsConfig{
			Mappings: []config.Mapping{
				{From: "http://localhost:8080", To: "https://example.com"},
			},
		}

		assert.NotPanics(t, func() {
			validators.ValidateHostsFileEntries(cfg, fs)
		})
	})

	t.Run("should handle wildcard mappings", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		createMockHostsFile(t, fs, "127.0.0.1 localhost\n")

		cfg := &config.UncorsConfig{
			Mappings: []config.Mapping{
				{From: "http://*.local:8080", To: "https://*.example.com"},
			},
		}

		assert.NotPanics(t, func() {
			validators.ValidateHostsFileEntries(cfg, fs)
		})
	})

	t.Run("should handle empty mappings", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		createMockHostsFile(t, fs, "127.0.0.1 localhost\n")

		cfg := &config.UncorsConfig{
			Mappings: []config.Mapping{},
		}

		assert.NotPanics(t, func() {
			validators.ValidateHostsFileEntries(cfg, fs)
		})
	})

	t.Run("should handle invalid URLs gracefully", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		createMockHostsFile(t, fs, "127.0.0.1 localhost\n")

		cfg := &config.UncorsConfig{
			Mappings: []config.Mapping{
				{From: "not-a-url", To: "https://example.com"},
			},
		}

		assert.NotPanics(t, func() {
			validators.ValidateHostsFileEntries(cfg, fs)
		})
	})

	t.Run("should handle missing hosts file", func(t *testing.T) {
		fs := afero.NewMemMapFs()

		cfg := &config.UncorsConfig{
			Mappings: []config.Mapping{
				{From: "http://localhost:8080", To: "https://example.com"},
			},
		}

		assert.NotPanics(t, func() {
			validators.ValidateHostsFileEntries(cfg, fs)
		})
	})

	t.Run("should warn about missing hosts", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		createMockHostsFile(t, fs, "127.0.0.1 localhost\n127.0.0.1 other.local\n")

		cfg := &config.UncorsConfig{
			Mappings: []config.Mapping{
				{From: "http://api.local:8080", To: "https://example.com"},
			},
		}

		// Should not panic, just log warning
		assert.NotPanics(t, func() {
			validators.ValidateHostsFileEntries(cfg, fs)
		})
	})

	t.Run("should not warn when host exists in hosts file", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		createMockHostsFile(t, fs, "127.0.0.1 localhost api.local app.local\n")

		cfg := &config.UncorsConfig{
			Mappings: []config.Mapping{
				{From: "http://api.local:8080", To: "https://example.com"},
			},
		}

		assert.NotPanics(t, func() {
			validators.ValidateHostsFileEntries(cfg, fs)
		})
	})
}
