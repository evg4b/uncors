package log_test

import (
	"bytes"
	"testing"

	"github.com/evg4b/uncors/internal/log"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

const loggerResult = "this is test message"
const loggerFResult = "this is %s message"

func TestGlobalLogPackage(t *testing.T) {
	log.DisableColor()

	t.Run("error logging", func(t *testing.T) {
		t.Run("Error", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
			log.Error(loggerResult)

			assert.Equal(t, "   ERROR  this is test message\n", output.String())
		}))

		t.Run("Errorf", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
			log.Errorf(loggerFResult, "error")

			assert.Equal(t, "   ERROR  this is error message\n", output.String())
		}))
	})

	t.Run("warning logging", func(t *testing.T) {
		t.Run("Warning", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
			log.Warning(loggerResult)

			assert.Equal(t, " WARNING  this is test message\n", output.String())
		}))

		t.Run("Warningf", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
			log.Warningf(loggerFResult, "warning")

			assert.Equal(t, " WARNING  this is warning message\n", output.String())
		}))
	})

	t.Run("info logging", func(t *testing.T) {
		t.Run("Info", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
			log.Info(loggerResult)

			assert.Equal(t, "    INFO  this is test message\n", output.String())
		}))

		t.Run("Infof", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
			log.Infof(loggerFResult, "info")

			assert.Equal(t, "    INFO  this is info message\n", output.String())
		}))
	})

	t.Run("debug logging", func(t *testing.T) {
		t.Run("where debug output enabled", func(t *testing.T) {
			log.EnableDebugMessages()

			t.Run("Debug", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
				log.Debug(loggerResult)

				assert.Equal(t, "   DEBUG  this is test message\n", output.String())
			}))

			t.Run("Debugf", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
				log.Debugf(loggerFResult, "debug")

				assert.Equal(t, "   DEBUG  this is debug message\n", output.String())
			}))
		})

		t.Run("where debug output disabled", func(t *testing.T) {
			log.DisableDebugMessages()

			t.Run("Debug", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
				log.Debug(loggerResult)

				assert.Empty(t, output.String())
			}))

			t.Run("Debugf", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
				log.Debugf(loggerFResult, "debug")

				assert.Empty(t, output.String())
			}))
		})
	})

	t.Run("raw printing", func(t *testing.T) {
		t.Run("Print", testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
			log.Print(loggerResult)

			assert.Equal(t, loggerResult, output.String())
		}))
	})
}

func TestColoring(t *testing.T) {
	t.Run(
		"should print additional tags where colors enabled",
		testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
			log.EnableColor()

			log.Error("test error")

			expected := "\x1b[30;101m\x1b[30;101m   ERROR \x1b[0m\x1b[0m \x1b[91m\x1b[91mtest error\x1b[0m\x1b[0m\n"
			assert.Equal(t, expected, output.String())
		}),
	)

	t.Run(
		"should print additional tags where colors disabled",
		testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
			log.DisableColor()

			log.Error("test error")

			expected := "   ERROR  test error\n"
			assert.Equal(t, expected, output.String())
		}),
	)
}

func TestOutputDisabling(t *testing.T) {
	log.DisableColor()

	t.Run(
		"should print log where output enabled",
		testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
			log.EnableOutput()

			log.Error(loggerResult)

			assert.Equal(t, "   ERROR  this is test message\n", output.String())
		}),
	)

	t.Run(
		"should not print log where output disabled",
		testutils.LogTest(func(t *testing.T, output *bytes.Buffer) {
			log.DisableOutput()

			log.Error(loggerResult)

			assert.Empty(t, output.String())
		}),
	)
}
