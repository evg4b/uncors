package main

import (
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/evg4b/uncors/internal/cli"
	"github.com/evg4b/uncors/internal/telemetry"
	"github.com/stretchr/testify/assert"
)

// main sets up telemetry, which replaces the default slog logger.
func saveTelemetry(t *testing.T) {
	t.Helper()

	t.Setenv(telemetry.EnvVar, "")

	orig := slog.Default()

	t.Cleanup(func() { slog.SetDefault(orig) })
}

func setArgs(t *testing.T, args []string) {
	t.Helper()

	orig := os.Args
	os.Args = args

	t.Cleanup(func() { os.Args = orig })
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
	saveTelemetry(t)
	setArgs(t, []string{"uncors", "--version"})

	assert.NotPanics(t, func() {
		main()
	})
}

func TestMain_GenerateCertsPath(t *testing.T) {
	saveTelemetry(t)
	t.Setenv("HOME", t.TempDir())
	setArgs(t, []string{"uncors", cli.GenerateCertsCmd, "--validity-days=7"})

	assert.NotPanics(t, func() {
		main()
	})
}

func TestMain_GenerateCertsHelpPath(t *testing.T) {
	saveTelemetry(t)
	setArgs(t, []string{"uncors", cli.GenerateCertsCmd, "--help"})

	assert.NotPanics(t, func() {
		main()
	})
}
