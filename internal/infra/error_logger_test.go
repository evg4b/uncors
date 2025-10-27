package infra_test

import (
	"bytes"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/stretchr/testify/assert"
)

func TestNewHTTPServerErrorLogger(t *testing.T) {
	t.Run("should log messages with ERROR level", func(t *testing.T) {
		var buf bytes.Buffer
		logger := log.New(&buf)
		logger.SetLevel(log.DebugLevel)

		stdLogger := infra.NewHTTPServerErrorLogger(logger)

		stdLogger.Print("test error message")

		output := buf.String()
		assert.Contains(t, output, "test error message")
		assert.Contains(t, output, "ERRO")
	})

	t.Run("should trim whitespace from messages", func(t *testing.T) {
		var buf bytes.Buffer
		logger := log.New(&buf)
		logger.SetLevel(log.DebugLevel)

		stdLogger := infra.NewHTTPServerErrorLogger(logger)

		stdLogger.Print("  test message with spaces  \n")

		output := buf.String()
		assert.Contains(t, output, "test message with spaces")
		assert.NotContains(t, output, "  test")
	})

	t.Run("should not log empty messages", func(t *testing.T) {
		var buf bytes.Buffer
		logger := log.New(&buf)
		logger.SetLevel(log.DebugLevel)

		stdLogger := infra.NewHTTPServerErrorLogger(logger)

		stdLogger.Print("   \n  \t  ")

		output := buf.String()
		assert.Empty(t, output)
	})
}
