package validators_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/config/validators"
	infratls "github.com/evg4b/uncors/internal/infra/tls"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateTLS(t *testing.T) {
	t.Run("skip validation for invalid URL", func(t *testing.T) {
		var errs validators.Errors
		validators.ValidateTLS(
			"test",
			config.Mapping{From: "://invalid-url", To: hosts.Example.HTTP()},
			afero.NewMemMapFs(),
			&errs,
		)
		assert.False(t, errs.HasAny())
	})

	t.Run("skip validation for non-HTTPS", func(t *testing.T) {
		var errs validators.Errors
		validators.ValidateTLS(
			"test",
			config.Mapping{From: "http://localhost:8080", To: hosts.Example.HTTP()},
			afero.NewMemMapFs(),
			&errs,
		)
		assert.False(t, errs.HasAny())
	})

	t.Run("error when CA does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		fakeHome := filepath.Join(tmpDir, "home")
		require.NoError(t, os.MkdirAll(fakeHome, 0o755))
		t.Setenv("HOME", fakeHome)

		var errs validators.Errors
		validators.ValidateTLS("test",
			config.Mapping{From: "https://localhost:8443", To: hosts.Example.HTTP()},
			afero.NewOsFs(), &errs)

		assert.True(t, errs.HasAny())
		assert.Contains(t, errs.Error(), "HTTPS mapping 'localhost:8443' requires a local CA certificate")
		assert.Contains(t, errs.Error(), "uncors generate-certs")
	})

	t.Run("pass when CA exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		fakeHome := filepath.Join(tmpDir, "home")
		require.NoError(t, os.MkdirAll(fakeHome, 0o755))
		t.Setenv("HOME", fakeHome)

		fs := afero.NewOsFs()
		caDir := filepath.Join(fakeHome, ".config", "uncors")
		_, _, err := infratls.GenerateCA(infratls.CAConfig{ValidityDays: 365, OutputDir: caDir, Fs: fs})
		require.NoError(t, err)

		var errs validators.Errors
		validators.ValidateTLS("test",
			config.Mapping{From: "https://localhost:8443", To: hosts.Example.HTTP()},
			fs, &errs)

		assert.False(t, errs.HasAny())
	})
}
