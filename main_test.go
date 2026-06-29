package main

import (
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/evg4b/uncors/internal/cli"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// saveLogger captures the current log writer and restores it after the test.
func saveLogger(t *testing.T) {
	t.Helper()

	orig := log.Writer()

	t.Cleanup(func() { log.SetOutput(orig) })
}

// setArgs temporarily overrides os.Args and restores it via t.Cleanup.
func setArgs(t *testing.T, args []string) {
	t.Helper()

	orig := os.Args
	os.Args = args

	t.Cleanup(func() { os.Args = orig })
}

func TestSetupLogging(t *testing.T) {
	t.Run("discards output when UNCORS_LOGGING is empty", func(t *testing.T) {
		saveLogger(t)
		t.Setenv("UNCORS_LOGGING", "")

		infra.SetupLogging()

		assert.Equal(t, io.Discard, log.Writer())
	})

	t.Run("writes to file when UNCORS_LOGGING points to a valid path", func(t *testing.T) {
		saveLogger(t)
		logPath := filepath.Join(t.TempDir(), "test.log")
		t.Setenv("UNCORS_LOGGING", logPath)

		infra.SetupLogging()

		require.NotEqual(t, io.Discard, log.Writer())

		_, err := os.Stat(logPath)
		assert.NoError(t, err)
	})

	t.Run("discards output when log file cannot be opened", func(t *testing.T) {
		saveLogger(t)
		t.Setenv("UNCORS_LOGGING", "/no-such-dir/test.log")

		infra.SetupLogging()

		assert.Equal(t, io.Discard, log.Writer())
	})
}

var errTest = errors.New("something went wrong")

func TestHandleError_ExitsOnError(t *testing.T) {
	orig := osExit

	var capturedCode int

	osExit = func(code int) { capturedCode = code }

	t.Cleanup(func() { osExit = orig })

	handleError(errTest)

	assert.Equal(t, 1, capturedCode)
}

func TestHandleError_NoopOnNil(t *testing.T) {
	orig := osExit

	called := false

	osExit = func(_ int) { called = true }

	t.Cleanup(func() { osExit = orig })

	handleError(nil)

	assert.False(t, called)
}

func TestMain_RunUncorsVersionPath(t *testing.T) {
	saveLogger(t)
	setArgs(t, []string{"uncors", "--version"})

	assert.NotPanics(t, func() {
		main()
	})
}

func TestMain_GenerateCertsPath(t *testing.T) {
	saveLogger(t)
	// Point HOME to a temp dir so CA certificates go there, not ~/.config/uncors.
	t.Setenv("HOME", t.TempDir())
	setArgs(t, []string{"uncors", cli.GenerateCertsCmd, "--validity-days=7"})

	assert.NotPanics(t, func() {
		main()
	})
}

func TestMain_GenerateCertsHelpPath(t *testing.T) {
	saveLogger(t)
	setArgs(t, []string{"uncors", cli.GenerateCertsCmd, "--help"})

	assert.NotPanics(t, func() {
		main()
	})
}
