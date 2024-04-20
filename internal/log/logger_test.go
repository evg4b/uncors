// nolint: bodyclose, goconst
package log_test

import (
	"bytes"
	"testing"

	"github.com/evg4b/uncors/internal/log"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/pterm/pterm"
	"github.com/stretchr/testify/assert"
)

const (
	testMessage  = "this is test message"
	testFMessage = "this is %s message"
	prefix       = " Test "
)

func TestPrefixedLogger(t *testing.T) {
	log.EnableOutput()
	log.DisableColor()
	log.EnableDebugMessages()

	t.Run("prefix printing", func(t *testing.T) {
		logger := log.NewLogger(prefix)

		t.Run("Error", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
			logger.Error(testMessage)

			assert.Equal(t, "  Test      ERROR  this is test message\n", output.String())
		}))

		t.Run("Errorf", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
			logger.Errorf(testFMessage, "Error")

			assert.Equal(t, "  Test      ERROR  this is Error message\n", output.String())
		}))

		t.Run("Info", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
			logger.Info(testMessage)

			assert.Equal(t, "  Test       INFO  this is test message\n", output.String())
		}))

		t.Run("Infof", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
			logger.Infof(testFMessage, "Info")

			assert.Equal(t, "  Test       INFO  this is Info message\n", output.String())
		}))

		t.Run("Warning", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
			logger.Warning(testMessage)

			assert.Equal(t, "  Test    WARNING  this is test message\n", output.String())
		}))

		t.Run("Warningf", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
			logger.Warningf(testFMessage, "Warning")

			assert.Equal(t, "  Test    WARNING  this is Warning message\n", output.String())
		}))

		t.Run("Debug", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
			logger.Debug(testMessage)

			assert.Equal(t, "  Test      DEBUG  this is test message\n", output.String())
		}))

		t.Run("Debugf", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
			logger.Debugf(testFMessage, "Debug")

			assert.Equal(t, "  Test      DEBUG  this is Debug message\n", output.String())
		}))
	})

	t.Run("custom output", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
		buffer := &bytes.Buffer{}

		logger := log.NewLogger(prefix, log.WithOutput(buffer))

		logger.Info("Test message")

		assert.Empty(t, output.String())
		assert.Equal(t, "  Test       INFO  Test message\n", buffer.String())
	}))

	t.Run("custom styles", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
		log.EnableColor()

		logger := log.NewLogger(prefix, log.WithStyle(&pterm.Style{
			pterm.BgBlue,
			pterm.BgLightBlue,
		}))

		logger.Info("Test message")

		expected := "\x1b[44;104m\x1b[44;104m  Test  \x1b[0m\x1b[0m \x1b[39;49m\x1b[39;49m\x1b[30;46m\x1b[30;46m" +
			"    INFO \x1b[0m\x1b[39;49m\x1b[0m\x1b[39;49m \x1b[96m\x1b[96mTest message\x1b[0m\x1b[39;49m\x1b[0m" +
			"\x1b[39;49m\x1b[0m\x1b[0m\n"

		assert.Equal(t, expected, output.String())
	}))
}
