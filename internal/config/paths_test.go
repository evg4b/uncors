package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The documentation uses `~/...` for every path-taking field. The shell expands
// `~` on the command line, but not inside the YAML file, which is exactly where
// the docs use it most.
func TestConfigurationPathsAreResolved(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	const configContent = `
mappings:
  - from: http://localhost:3000
    to: https://api.example.com
    har: ./recordings/api.har
    statics:
      - path: /
        dir: ~/projects/app/dist
        index: index.html
    mocks:
      - path: /api/users
        response:
          code: 200
          file: ./mocks/users.json
    scripts:
      - path: /api/script
        file: /absolute/script.lua
`

	fs := afero.NewMemMapFs()

	configDir := filepath.Join("/workspace", "project")
	configPath := filepath.Join(configDir, ".uncors.yaml")

	require.NoError(t, afero.WriteFile(fs, configPath, []byte(configContent), 0o600))
	require.NoError(t, fs.MkdirAll(filepath.Join(home, "projects/app/dist"), 0o755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(home, "projects/app/dist/index.html"), []byte(""), 0o600))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(configDir, "mocks/users.json"), []byte("{}"), 0o600))
	require.NoError(t, afero.WriteFile(fs, "/absolute/script.lua", []byte(""), 0o600))

	flags, err := config.ParseFlags([]string{"--config", configPath})
	require.NoError(t, err)

	cfg, err := config.LoadConfiguration(fs, flags)
	require.NoError(t, err)

	mapping := cfg.Mappings[0]

	assert.Equal(t, filepath.Join(home, "projects/app/dist"), mapping.Statics[0].Dir,
		"~ expands to the home directory")
	assert.Equal(t, filepath.Join(configDir, "mocks/users.json"), mapping.Mocks[0].Response.File,
		"a relative path resolves against the config file, not the working directory")
	assert.Equal(t, filepath.Join(configDir, "recordings/api.har"), mapping.HAR.File)
	assert.Equal(t, "/absolute/script.lua", mapping.Scripts[0].File,
		"an absolute path is used as it is")
}
