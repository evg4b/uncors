package infra

import (
	"log"
	"strings"

	clog "github.com/charmbracelet/log"
)

type errorLogWriter struct {
	logger *clog.Logger
}

// Write implements io.Writer interface for HTTP server error logging.
func (w *errorLogWriter) Write(data []byte) (int, error) {
	msg := strings.TrimSpace(string(data))
	if msg != "" {
		w.logger.Error(msg)
	}

	return len(data), nil
}

// NewHTTPServerErrorLogger creates a standard Go logger that outputs to charmbracelet log with ERROR level.
func NewHTTPServerErrorLogger(logger *clog.Logger) *log.Logger {
	return log.New(&errorLogWriter{logger: logger}, "", 0)
}
