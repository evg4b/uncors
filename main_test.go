package main

import (
	"context"
	"os"
	"testing"

	"github.com/evg4b/uncors/internal/di"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setArgs temporarily overrides os.Args and returns a restore function.
func setArgs(args []string) func() {
	old := os.Args
	os.Args = args

	return func() { os.Args = old }
}

func TestLoadConfiguration(t *testing.T) {
	t.Run("returns config for valid flags", func(t *testing.T) {
		defer setArgs([]string{"uncors", "-f", "http://localhost:3000", "-t", "https://api.example.com"})()

		cfg, path, err := loadConfiguration(afero.NewMemMapFs())

		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Empty(t, path)
		assert.Len(t, cfg.Mappings, 1)
	})

	t.Run("fails when mappings are empty", func(t *testing.T) {
		defer setArgs([]string{"uncors"})()

		_, _, err := loadConfiguration(afero.NewMemMapFs())

		require.Error(t, err)
	})

	t.Run("fails on invalid flags", func(t *testing.T) {
		defer setArgs([]string{"uncors", "--no-such-flag"})()

		_, _, err := loadConfiguration(afero.NewMemMapFs())

		require.Error(t, err)
	})
}

func TestRunGenerateCerts(t *testing.T) {
	t.Run("generates certs and returns 0", func(t *testing.T) {
		defer setArgs([]string{"uncors", generateCertsCmd})()

		container := di.NewContainer()
		defer testutils.Close(t, container)

		result := runGenerateCerts(container)

		assert.Equal(t, 0, result)
	})

	t.Run("returns 1 when execute fails", func(t *testing.T) {
		defer setArgs([]string{"uncors", generateCertsCmd})()

		container := di.NewContainer()
		defer testutils.Close(t, container)

		_ = runGenerateCerts(container)
		result := runGenerateCerts(container)

		assert.Equal(t, 1, result)
	})

	t.Run("returns 1 when flags parse fails", func(t *testing.T) {
		defer setArgs([]string{"uncors", generateCertsCmd, "--no-such-flag"})()

		container := di.NewContainer()
		defer testutils.Close(t, container)

		result := runGenerateCerts(container)

		assert.Equal(t, 1, result)
	})
}

func TestLoadConfigurationWithDebug(t *testing.T) {
	t.Chdir(t.TempDir())

	defer setArgs([]string{"uncors", "-f", "http://localhost:3000", "-t", "https://api.example.com", "--debug"})()

	cfg, _, err := loadConfiguration(afero.NewMemMapFs())

	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.True(t, cfg.Debug)
}

func TestLoadConfigurationWithConfigFile(t *testing.T) {
	const cfgContent = `
mappings:
  - from: http://localhost:3000
    to: https://api.example.com
`

	defer setArgs([]string{"uncors", "--config", "/config.yaml"})()

	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "/config.yaml", []byte(cfgContent), 0o600))

	cfg, path, err := loadConfiguration(fs)

	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "/config.yaml", path)
	assert.Len(t, cfg.Mappings, 1)
}

func TestStartVersionChecker(t *testing.T) {
	t.Run("runs without panic", func(t *testing.T) {
		container := di.NewContainer()
		defer testutils.Close(t, container)

		assert.NotPanics(t, func() {
			startVersionChecker(context.Background(), container, "")
		})
	})
}

func TestConfigLoader(t *testing.T) {
	t.Run("propagates configuration errors instead of returning a nil config", func(t *testing.T) {
		defer setArgs([]string{"uncors"})()

		cfg, err := configLoader(afero.NewMemMapFs())()

		require.Error(t, err)
		assert.Nil(t, cfg)
	})

	t.Run("returns the loaded configuration", func(t *testing.T) {
		defer setArgs([]string{"uncors", "-f", "http://localhost:3000", "-t", "https://api.example.com"})()

		cfg, err := configLoader(afero.NewMemMapFs())()

		require.NoError(t, err)
		require.NotNil(t, cfg)
	})
}
