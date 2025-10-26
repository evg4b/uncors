package validators_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/config/validators"
	infratls "github.com/evg4b/uncors/internal/infra/tls"
	"github.com/gobuffalo/validate"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTLSValidator_IsValid(t *testing.T) {
	t.Run("should skip validation for invalid URL", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		mapping := config.Mapping{
			From: "://invalid-url",
			To:   "http://example.com",
		}

		validator := &validators.TLSValidator{
			Field:   "test",
			Mapping: mapping,
			Fs:      fs,
		}

		errors := validate.NewErrors()
		validator.IsValid(errors)

		assert.False(t, errors.HasAny())
	})

	t.Run("should skip validation for non-HTTPS schemes", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		mapping := config.Mapping{
			From: "http://localhost:8080",
			To:   "http://example.com",
		}

		validator := &validators.TLSValidator{
			Field:   "test",
			Mapping: mapping,
			Fs:      fs,
		}

		errors := validate.NewErrors()
		validator.IsValid(errors)

		assert.False(t, errors.HasAny())
	})

	t.Run("should check CA availability for HTTPS mappings", func(t *testing.T) {
		tmpDir := t.TempDir()

		fakeHome := filepath.Join(tmpDir, "home")
		require.NoError(t, os.MkdirAll(fakeHome, 0o755))
		t.Setenv("HOME", fakeHome)

		fs := afero.NewOsFs()

		mapping := config.Mapping{
			From: "https://localhost:8443",
			To:   "http://example.com",
		}

		validator := &validators.TLSValidator{
			Field:   "test",
			Mapping: mapping,
			Fs:      fs,
		}

		errors := validate.NewErrors()
		validator.IsValid(errors)

		// CA doesn't exist, should error
		assert.True(t, errors.HasAny())
		errorMsg := errors.Get("test")[0]
		assert.Contains(t, errorMsg, "HTTPS mapping 'localhost:8443' requires a local CA certificate")
		assert.Contains(t, errorMsg, "uncors generate-certs")
	})

	t.Run("should pass when CA exists", func(t *testing.T) {
		tmpDir := t.TempDir()

		fakeHome := filepath.Join(tmpDir, "home")
		require.NoError(t, os.MkdirAll(fakeHome, 0o755))
		t.Setenv("HOME", fakeHome)

		fs := afero.NewOsFs()

		// Generate CA
		caDir := filepath.Join(fakeHome, ".config", "uncors")
		caConfig := infratls.CAConfig{
			ValidityDays: 365,
			OutputDir:    caDir,
			Fs:           fs,
		}
		_, _, err := infratls.GenerateCA(caConfig)
		require.NoError(t, err)

		mapping := config.Mapping{
			From: "https://localhost:8443",
			To:   "http://example.com",
		}

		validator := &validators.TLSValidator{
			Field:   "test",
			Mapping: mapping,
			Fs:      fs,
		}

		errors := validate.NewErrors()
		validator.IsValid(errors)

		// CA exists, should pass
		assert.False(t, errors.HasAny())
	})
}
