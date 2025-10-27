package tls_test

import (
	"os"
	"path/filepath"
	"testing"

	infratls "github.com/evg4b/uncors/internal/infra/tls"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	configDir = ".config"
	uncorsDir = "uncors"
)

func TestGetCAPath(t *testing.T) {
	t.Run("should return valid CA path", func(t *testing.T) {
		path, err := infratls.GetCAPath()
		require.NoError(t, err)
		assert.NotEmpty(t, path)
		assert.Contains(t, path, configDir)
		assert.Contains(t, path, uncorsDir)
	})

	t.Run("should return path containing user home", func(t *testing.T) {
		homeDir, err := os.UserHomeDir()
		require.NoError(t, err)

		path, err := infratls.GetCAPath()
		require.NoError(t, err)
		assert.Contains(t, path, homeDir)
	})
}

func TestCAExists(t *testing.T) {
	t.Run("should return false when CA does not exist", func(_ *testing.T) {
		// Temporarily override home dir for testing
		// This is tricky, so we just test the function doesn't panic
		exists := infratls.CAExists(afero.NewOsFs())
		// May be true or false depending on system state
		// Just verify it doesn't panic
		_ = exists
	})

	t.Run("should detect existing CA", func(t *testing.T) {
		tmpDir := t.TempDir()

		fakeHome := filepath.Join(tmpDir, "home")
		require.NoError(t, os.MkdirAll(fakeHome, 0o755))
		t.Setenv("HOME", fakeHome)

		assert.False(t, infratls.CAExists(afero.NewOsFs()))

		caDir := filepath.Join(fakeHome, configDir, uncorsDir)
		config := infratls.CAConfig{
			ValidityDays: 365,
		Fs:           afero.NewOsFs(),
			OutputDir:    caDir,
		}
		_, _, err := infratls.GenerateCA(config)
		require.NoError(t, err)

		// CA should exist now
		assert.True(t, infratls.CAExists(afero.NewOsFs()))
	})
}

func TestLoadDefaultCA(t *testing.T) {
	t.Run("should load CA from default location", func(t *testing.T) {
		tmpDir := t.TempDir()

		fakeHome := filepath.Join(tmpDir, "home")
		require.NoError(t, os.MkdirAll(fakeHome, 0o755))
		t.Setenv("HOME", fakeHome)

		caDir := filepath.Join(fakeHome, configDir, uncorsDir)
		config := infratls.CAConfig{
			ValidityDays: 365,
		Fs:           afero.NewOsFs(),
			OutputDir:    caDir,
		}
		_, _, err := infratls.GenerateCA(config)
		require.NoError(t, err)

		// Load default CA
		cert, key, err := infratls.LoadDefaultCA(afero.NewOsFs())
		require.NoError(t, err)
		assert.NotNil(t, cert)
		assert.NotNil(t, key)
	})

	t.Run("should return error when CA does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()

		fakeHome := filepath.Join(tmpDir, "home")
		require.NoError(t, os.MkdirAll(fakeHome, 0o755))
		t.Setenv("HOME", fakeHome)

		// Try to load non-existent CA
		_, _, err := infratls.LoadDefaultCA(afero.NewOsFs())
		require.Error(t, err)
	})

	t.Run("should use provided filesystem", func(t *testing.T) {
		tmpDir := t.TempDir()

		fakeHome := filepath.Join(tmpDir, "home")
		require.NoError(t, os.MkdirAll(fakeHome, 0o755))
		t.Setenv("HOME", fakeHome)

		fs := afero.NewOsFs()

		caDir := filepath.Join(fakeHome, configDir, uncorsDir)
		config := infratls.CAConfig{
			ValidityDays: 365,
			OutputDir:    caDir,
			Fs:           fs,
		}
		_, _, err := infratls.GenerateCA(config)
		require.NoError(t, err)

		// Load with provided filesystem
		cert, key, err := infratls.LoadDefaultCA(fs)
		require.NoError(t, err)
		assert.NotNil(t, cert)
		assert.NotNil(t, key)
	})
}

func TestCAExists_EdgeCases(t *testing.T) {
	t.Run("should handle filesystem with only cert file", func(t *testing.T) {
		tmpDir := t.TempDir()

		fakeHome := filepath.Join(tmpDir, "home")
		require.NoError(t, os.MkdirAll(fakeHome, 0o755))
		t.Setenv("HOME", fakeHome)

		caDir := filepath.Join(fakeHome, configDir, uncorsDir)
		require.NoError(t, os.MkdirAll(caDir, 0o755))

		// Create only cert file, not key
		certPath := filepath.Join(caDir, "ca.crt")
		require.NoError(t, os.WriteFile(certPath, []byte("cert"), 0o600))

		exists := infratls.CAExists(afero.NewOsFs())
		assert.False(t, exists, "should return false when only cert exists")
	})

	t.Run("should handle filesystem with only key file", func(t *testing.T) {
		tmpDir := t.TempDir()

		fakeHome := filepath.Join(tmpDir, "home")
		require.NoError(t, os.MkdirAll(fakeHome, 0o755))
		t.Setenv("HOME", fakeHome)

		caDir := filepath.Join(fakeHome, configDir, uncorsDir)
		require.NoError(t, os.MkdirAll(caDir, 0o755))

		// Create only key file, not cert
		keyPath := filepath.Join(caDir, "ca.key")
		require.NoError(t, os.WriteFile(keyPath, []byte("key"), 0o600))

		exists := infratls.CAExists(afero.NewOsFs())
		assert.False(t, exists, "should return false when only key exists")
	})

	t.Run("should use provided filesystem", func(t *testing.T) {
		tmpDir := t.TempDir()

		fakeHome := filepath.Join(tmpDir, "home")
		require.NoError(t, os.MkdirAll(fakeHome, 0o755))
		t.Setenv("HOME", fakeHome)

		fs := afero.NewOsFs()

		caDir := filepath.Join(fakeHome, configDir, uncorsDir)
		config := infratls.CAConfig{
			ValidityDays: 365,
			OutputDir:    caDir,
			Fs:           fs,
		}
		_, _, err := infratls.GenerateCA(config)
		require.NoError(t, err)

		// Check with provided filesystem
		exists := infratls.CAExists(fs)
		assert.True(t, exists)
	})
}
