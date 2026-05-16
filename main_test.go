package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/evg4b/uncors/internal/tui"
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

func newTestOutput() *tui.CliOutput {
	return tui.NewCliOutput(io.Discard)
}

func TestLoadConfiguration(t *testing.T) {
	t.Run("returns config for valid flags", func(t *testing.T) {
		defer setArgs([]string{"uncors", "-f", "http://localhost:3000", "-t", "https://api.example.com"})()

		cfg, path := loadConfiguration(afero.NewMemMapFs())

		require.NotNil(t, cfg)
		assert.Empty(t, path)
		assert.Len(t, cfg.Mappings, 1)
	})

	t.Run("panics when mappings are empty", func(t *testing.T) {
		defer setArgs([]string{"uncors"})()

		assert.Panics(t, func() {
			loadConfiguration(afero.NewMemMapFs())
		})
	})

	t.Run("panics on invalid flags", func(t *testing.T) {
		defer setArgs([]string{"uncors", "--no-such-flag"})()

		assert.Panics(t, func() {
			loadConfiguration(afero.NewMemMapFs())
		})
	})
}

func TestRunGenerateCerts(t *testing.T) {
	t.Run("generates certs and returns 0", func(t *testing.T) {
		defer setArgs([]string{"uncors", generateCertsCmd})()

		fs := afero.NewMemMapFs()
		output := newTestOutput()

		result := runGenerateCerts(fs, output)

		assert.Equal(t, 0, result)
	})

	t.Run("returns 1 when execute fails", func(t *testing.T) {
		defer setArgs([]string{"uncors", generateCertsCmd})()

		// Second call on the same fs finds certs already exist → ErrCAAlreadyExists.
		fs := afero.NewMemMapFs()
		output := newTestOutput()

		_ = runGenerateCerts(fs, output)
		result := runGenerateCerts(fs, output)

		assert.Equal(t, 1, result)
	})
}

func TestLoadConfigurationWithDebug(t *testing.T) {
	t.Chdir(t.TempDir())

	defer setArgs([]string{"uncors", "-f", "http://localhost:3000", "-t", "https://api.example.com", "--debug"})()

	cfg, _ := loadConfiguration(afero.NewMemMapFs())

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

	cfg, path := loadConfiguration(fs)

	require.NotNil(t, cfg)
	assert.Equal(t, "/config.yaml", path)
	assert.Len(t, cfg.Mappings, 1)
}

func TestStartConfigWatcher(t *testing.T) {
	t.Run("logs error for non-existent config path", func(t *testing.T) {
		output := newTestOutput()

		assert.NotPanics(t, func() {
			startConfigWatcher(context.Background(), afero.NewMemMapFs(), output, "/no/such/config.yaml", nil)
		})
	})

	t.Run("creates watcher for existing config file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "config.yaml")
		require.NoError(t, os.WriteFile(configFile, []byte("proxy: \"\""), 0o600))

		output := newTestOutput()

		assert.NotPanics(t, func() {
			startConfigWatcher(context.Background(), afero.NewMemMapFs(), output, configFile, nil)
		})
	})
}
