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

	t.Run("should require both cert and key files together", func(t *testing.T) {
		fs := afero.NewMemMapFs()

		testCases := []struct {
			name     string
			certFile string
			keyFile  string
		}{
			{
				name:     "cert without key",
				certFile: "/path/to/cert.crt",
				keyFile:  "",
			},
			{
				name:     "key without cert",
				certFile: "",
				keyFile:  "/path/to/key.key",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				mapping := config.Mapping{
					From:     "https://localhost:8443",
					To:       "http://example.com",
					CertFile: tc.certFile,
					KeyFile:  tc.keyFile,
				}

				validator := &validators.TLSValidator{
					Field:   "test",
					Mapping: mapping,
					Fs:      fs,
				}

				errors := validate.NewErrors()
				validator.IsValid(errors)

				assert.True(t, errors.HasAny())
				assert.Contains(t, errors.Get("test")[0], "both cert-file and key-file must be provided together")
			})
		}
	})

	t.Run("should validate custom certificate files exist", func(t *testing.T) {
		fs := afero.NewMemMapFs()

		certPath := "/path/to/cert.crt"
		keyPath := "/path/to/key.key"

		// Create only cert file, not key
		require.NoError(t, afero.WriteFile(fs, certPath, []byte("cert"), 0o644))

		mapping := config.Mapping{
			From:     "https://localhost:8443",
			To:       "http://example.com",
			CertFile: certPath,
			KeyFile:  keyPath,
		}

		validator := &validators.TLSValidator{
			Field:   "test",
			Mapping: mapping,
			Fs:      fs,
		}

		errors := validate.NewErrors()
		validator.IsValid(errors)

		assert.True(t, errors.HasAny())
		assert.Contains(t, errors.Get("test.key-file")[0], "key file not found")
	})

	t.Run("should validate both certificate files exist", func(t *testing.T) {
		fs := afero.NewMemMapFs()

		certPath := "/path/to/cert.crt"
		keyPath := "/path/to/key.key"

		// Don't create files, both should error
		mapping := config.Mapping{
			From:     "https://localhost:8443",
			To:       "http://example.com",
			CertFile: certPath,
			KeyFile:  keyPath,
		}

		validator := &validators.TLSValidator{
			Field:   "test",
			Mapping: mapping,
			Fs:      fs,
		}

		errors := validate.NewErrors()
		validator.IsValid(errors)

		assert.True(t, errors.HasAny())
		assert.Contains(t, errors.Get("test.cert-file")[0], "certificate file not found")
		assert.Contains(t, errors.Get("test.key-file")[0], "key file not found")
	})

	t.Run("should pass when both certificate files exist", func(t *testing.T) {
		fs := afero.NewMemMapFs()

		certPath := "/path/to/cert.crt"
		keyPath := "/path/to/key.key"

		// Create both files
		require.NoError(t, afero.WriteFile(fs, certPath, []byte("cert"), 0o644))
		require.NoError(t, afero.WriteFile(fs, keyPath, []byte("key"), 0o600))

		mapping := config.Mapping{
			From:     "https://localhost:8443",
			To:       "http://example.com",
			CertFile: certPath,
			KeyFile:  keyPath,
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

	t.Run("should check CA availability when no custom certs provided", func(t *testing.T) {
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
		assert.Contains(t, errorMsg, "HTTPS mapping 'localhost:8443' requires TLS certificates")
		assert.Contains(t, errorMsg, "uncors generate-certs")
	})

	t.Run("should pass when CA exists and no custom certs provided", func(t *testing.T) {
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
