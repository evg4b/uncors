package config_test

import (
	"bytes"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadWithStatics(t *testing.T, statics string) string {
	t.Helper()

	restore := slog.Default()

	t.Cleanup(func() { slog.SetDefault(restore) })

	logs := &bytes.Buffer{}
	slog.SetDefault(slog.New(slog.NewTextHandler(logs, nil)))

	fs := afero.NewMemMapFs()
	configPath := "/project/.uncors.yaml"

	content := `
mappings:
  - from: http://localhost:3000
    to: https://api.example.com
    statics:
` + statics

	require.NoError(t, afero.WriteFile(fs, configPath, []byte(content), 0o600))
	require.NoError(t, fs.MkdirAll(filepath.Join("/project", "dist"), 0o755))
	require.NoError(t, fs.MkdirAll(filepath.Join("/project", "assets"), 0o755))

	flags, err := config.ParseFlags([]string{"--config", configPath})
	require.NoError(t, err)

	_, err = config.LoadConfiguration(fs, flags)
	require.NoError(t, err)

	return logs.String()
}

func TestShadowedStaticMountsAreReported(t *testing.T) {
	t.Run("reports a mount an earlier one already covers", func(t *testing.T) {
		logs := loadWithStatics(t, `      - path: /
        dir: ./dist
      - path: /assets
        dir: ./assets
`)

		assert.Contains(t, logs, "unreachable")
		assert.Contains(t, logs, "/assets")
	})

	t.Run("says nothing when the mounts do not overlap", func(t *testing.T) {
		logs := loadWithStatics(t, `      - path: /assets
        dir: ./assets
      - path: /
        dir: ./dist
`)

		assert.NotContains(t, logs, "unreachable")
	})
}
