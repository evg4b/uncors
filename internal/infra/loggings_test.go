package infra_test

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/evg4b/uncors/internal/infra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func saveLogWriter(t *testing.T) {
	t.Helper()

	orig := log.Writer()

	t.Cleanup(func() { log.SetOutput(orig) })
}

func TestSetupLogging(t *testing.T) {
	t.Run("discards output when UNCORS_LOGGING is empty", func(t *testing.T) {
		saveLogWriter(t)
		t.Setenv("UNCORS_LOGGING", "")

		infra.SetupLogging()

		assert.Equal(t, io.Discard, log.Writer())
	})

	t.Run("writes to file when UNCORS_LOGGING points to a valid path", func(t *testing.T) {
		saveLogWriter(t)
		logPath := filepath.Join(t.TempDir(), "test.log")
		t.Setenv("UNCORS_LOGGING", logPath)

		infra.SetupLogging()

		require.NotEqual(t, io.Discard, log.Writer())

		_, err := os.Stat(logPath)
		assert.NoError(t, err)
	})

	t.Run("discards output when log file cannot be opened", func(t *testing.T) {
		saveLogWriter(t)
		t.Setenv("UNCORS_LOGGING", "/no-such-dir/test.log")

		infra.SetupLogging()

		assert.Equal(t, io.Discard, log.Writer())
	})
}
