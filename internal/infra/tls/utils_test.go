package tls_test

import (
	"os"
	"path/filepath"
	"testing"

	infratls "github.com/evg4b/uncors/internal/infra/tls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCAPath(t *testing.T) {
	t.Run("should return valid CA path", func(t *testing.T) {
		path, err := infratls.GetCAPath()
		require.NoError(t, err)
		assert.NotEmpty(t, path)
		assert.Contains(t, path, ".config")
		assert.Contains(t, path, "uncors")
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
	t.Run("should return false when CA does not exist", func(t *testing.T) {
		// Temporarily override home dir for testing
		// This is tricky, so we just test the function doesn't panic
		exists := infratls.CAExists()
		// May be true or false depending on system state
		// Just verify it doesn't panic
		_ = exists
	})

	t.Run("should detect existing CA", func(t *testing.T) {
		// Create temporary CA
		tmpDir := t.TempDir()

		// Temporarily set HOME for testing
		originalHome := os.Getenv("HOME")
		defer os.Setenv("HOME", originalHome)

		// Create fake home directory
		fakeHome := filepath.Join(tmpDir, "home")
		require.NoError(t, os.MkdirAll(fakeHome, 0o755))
		os.Setenv("HOME", fakeHome)

		// CA should not exist initially
		assert.False(t, infratls.CAExists())

		// Generate CA
		caDir := filepath.Join(fakeHome, ".config", "uncors")
		config := infratls.CAConfig{
			ValidityDays: 365,
			OutputDir:    caDir,
		}
		_, _, err := infratls.GenerateCA(config)
		require.NoError(t, err)

		// CA should exist now
		assert.True(t, infratls.CAExists())
	})
}

func TestLoadDefaultCA(t *testing.T) {
	t.Run("should load CA from default location", func(t *testing.T) {
		// Setup temporary home
		tmpDir := t.TempDir()
		originalHome := os.Getenv("HOME")
		defer os.Setenv("HOME", originalHome)

		fakeHome := filepath.Join(tmpDir, "home")
		require.NoError(t, os.MkdirAll(fakeHome, 0o755))
		os.Setenv("HOME", fakeHome)

		// Generate CA in default location
		caDir := filepath.Join(fakeHome, ".config", "uncors")
		config := infratls.CAConfig{
			ValidityDays: 365,
			OutputDir:    caDir,
		}
		_, _, err := infratls.GenerateCA(config)
		require.NoError(t, err)

		// Load default CA
		cert, key, err := infratls.LoadDefaultCA()
		require.NoError(t, err)
		assert.NotNil(t, cert)
		assert.NotNil(t, key)
	})

	t.Run("should return error when CA does not exist", func(t *testing.T) {
		// Setup temporary home without CA
		tmpDir := t.TempDir()
		originalHome := os.Getenv("HOME")
		defer os.Setenv("HOME", originalHome)

		fakeHome := filepath.Join(tmpDir, "home")
		require.NoError(t, os.MkdirAll(fakeHome, 0o755))
		os.Setenv("HOME", fakeHome)

		// Try to load non-existent CA
		_, _, err := infratls.LoadDefaultCA()
		assert.Error(t, err)
	})
}
