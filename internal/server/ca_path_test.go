package server_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/evg4b/uncors/internal/server"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	configDir = ".config"
	uncorsDir = "uncors"
)

func fakeCADir(t *testing.T) string {
	t.Helper()

	fakeHome := filepath.Join(t.TempDir(), "home")
	require.NoError(t, os.MkdirAll(fakeHome, 0o755))
	t.Setenv("HOME", fakeHome)

	return filepath.Join(fakeHome, configDir, uncorsDir)
}

func generateDefaultCA(t *testing.T, caDir string) {
	t.Helper()

	_, _, err := server.GenerateCA(server.CAConfig{
		ValidityDays: 365,
		Fs:           afero.NewOsFs(),
		OutputDir:    caDir,
	})
	require.NoError(t, err)
}

func TestGetCAPath(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	path, err := server.GetCAPath()
	require.NoError(t, err)

	assert.Equal(t, filepath.Join(homeDir, configDir, uncorsDir), path)
}

func TestCAExists(t *testing.T) {
	t.Run("detects a generated CA", func(t *testing.T) {
		caDir := fakeCADir(t)

		assert.False(t, server.CAExists(afero.NewOsFs()))

		generateDefaultCA(t, caDir)

		assert.True(t, server.CAExists(afero.NewOsFs()))
	})

	for _, file := range []string{"ca.crt", "ca.key"} {
		t.Run("false when only "+file+" exists", func(t *testing.T) {
			caDir := fakeCADir(t)
			require.NoError(t, os.MkdirAll(caDir, 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(caDir, file), []byte(file), 0o600))

			assert.False(t, server.CAExists(afero.NewOsFs()))
		})
	}
}

func TestLoadDefaultCA(t *testing.T) {
	t.Run("loads CA from default location", func(t *testing.T) {
		generateDefaultCA(t, fakeCADir(t))

		cert, key, err := server.LoadDefaultCA(afero.NewOsFs())
		require.NoError(t, err)
		assert.NotNil(t, cert)
		assert.NotNil(t, key)
	})

	t.Run("returns error when CA does not exist", func(t *testing.T) {
		fakeCADir(t)

		_, _, err := server.LoadDefaultCA(afero.NewOsFs())
		require.Error(t, err)
	})
}
