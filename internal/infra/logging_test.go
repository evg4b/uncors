package infra_test

import (
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/evg4b/uncors/internal/infra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLogLevel(t *testing.T) {
	levels := map[string]slog.Level{
		"":        slog.LevelInfo,
		"info":    slog.LevelInfo,
		"DEBUG":   slog.LevelDebug,
		"warn":    slog.LevelWarn,
		"warning": slog.LevelWarn,
		"error":   slog.LevelError,
	}

	for name, expected := range levels {
		t.Run(name, func(t *testing.T) {
			level, err := infra.ParseLogLevel(name)

			require.NoError(t, err)
			assert.Equal(t, expected, level)
		})
	}

	t.Run("reports an unknown level", func(t *testing.T) {
		_, err := infra.ParseLogLevel("chatty")

		require.ErrorIs(t, err, infra.ErrUnknownLogLevel)
	})
}

func TestSetupLogging(t *testing.T) {
	restore := slog.Default()

	t.Cleanup(func() { slog.SetDefault(restore) })

	t.Run("writes diagnostics to the requested file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "uncors.log")

		closer, err := infra.SetupLogging(infra.LogOptions{Level: "debug", File: path})
		require.NoError(t, err)
		require.NotNil(t, closer)

		slog.Debug("hello from slog")
		// The standard log package is routed through the same handler, so
		// nothing that still uses it is lost.
		log.Print("hello from log")

		require.NoError(t, closer.Close())

		content, err := os.ReadFile(path) //nolint:gosec // a path this test created
		require.NoError(t, err)

		assert.Contains(t, string(content), "hello from slog")
		assert.Contains(t, string(content), "hello from log")
	})

	t.Run("keeps records below the level out", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "uncors.log")

		closer, err := infra.SetupLogging(infra.LogOptions{Level: "error", File: path})
		require.NoError(t, err)

		slog.Info("not important enough")
		slog.Error("important")

		require.NoError(t, closer.Close())

		content, err := os.ReadFile(path) //nolint:gosec // a path this test created
		require.NoError(t, err)

		assert.NotContains(t, string(content), "not important enough")
		assert.Contains(t, string(content), "important")
	})

	t.Run("reports an unusable log file", func(t *testing.T) {
		_, err := infra.SetupLogging(infra.LogOptions{File: filepath.Join(t.TempDir(), "no", "such", "dir", "x.log")})

		require.Error(t, err)
	})

	t.Run("provides a std logger for net/http", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "uncors.log")

		closer, err := infra.SetupLogging(infra.LogOptions{Level: "warn", File: path})
		require.NoError(t, err)

		infra.StdLogger().Print("http server error")

		require.NoError(t, closer.Close())

		content, err := os.ReadFile(path) //nolint:gosec // a path this test created
		require.NoError(t, err)

		assert.Contains(t, string(content), "http server error")
	})
}
