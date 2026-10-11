package logging_test

import (
	"log"
	"log/slog"
	"testing"

	"github.com/evg4b/uncors/internal/logging"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const logPath = "/logs/uncors.log"

func restoreDefault(t *testing.T) {
	t.Helper()

	orig := slog.Default()
	origFlags := log.Flags()

	t.Cleanup(func() {
		slog.SetDefault(orig)
		log.SetFlags(origFlags)
	})
}

func TestSetup(t *testing.T) {
	t.Run("writes slog and log records to the file", func(t *testing.T) {
		restoreDefault(t)

		fs := afero.NewMemMapFs()
		require.NoError(t, fs.MkdirAll("/logs", 0o755))

		logger, closer := logging.Setup(fs, logPath)

		logger.Debug("debug record", "key", "value")
		slog.Info("default record")
		log.Print("std log record")
		testutils.Close(t, closer)

		content, err := afero.ReadFile(fs, logPath)
		require.NoError(t, err)

		assert.Contains(t, string(content), "level=DEBUG msg=\"debug record\" key=value")
		assert.Contains(t, string(content), "msg=\"default record\"")
		assert.Contains(t, string(content), "std log record")
	})

	t.Run("appends to an existing file", func(t *testing.T) {
		restoreDefault(t)

		fs := afero.NewMemMapFs()
		require.NoError(t, afero.WriteFile(fs, logPath, []byte("previous run\n"), 0o644))

		logger, closer := logging.Setup(fs, logPath)
		logger.Info("next run")
		testutils.Close(t, closer)

		content, err := afero.ReadFile(fs, logPath)
		require.NoError(t, err)

		assert.Contains(t, string(content), "previous run\n")
		assert.Contains(t, string(content), "next run")
	})

	t.Run("discards records when the path is empty", func(t *testing.T) {
		restoreDefault(t)

		logger, closer := logging.Setup(afero.NewMemMapFs(), "")
		defer testutils.Close(t, closer)

		assert.False(t, logger.Enabled(t.Context(), slog.LevelError))
		assert.Same(t, logger, slog.Default())
	})

	t.Run("discards records when the file cannot be opened", func(t *testing.T) {
		restoreDefault(t)

		fs := afero.NewReadOnlyFs(afero.NewMemMapFs())

		logger, closer := logging.Setup(fs, logPath)
		defer testutils.Close(t, closer)

		assert.False(t, logger.Enabled(t.Context(), slog.LevelError))
	})
}
