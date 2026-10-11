// Package logging sets up the process-wide diagnostic log.
//
// The log is for troubleshooting UNCORS itself and never reaches the terminal:
// only the frontends (the interactive UI and the headless printer) write there.
// Every record goes through one slog.Handler, so exporting logs elsewhere (for
// example to OpenTelemetry) means replacing or wrapping that handler here.
package logging

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
)

// EnvVar names the environment variable holding the log file path.
const EnvVar = "UNCORS_LOGGING"

const (
	logFileFlags = os.O_CREATE | os.O_WRONLY | os.O_APPEND
	logFilePerm  = 0o644
)

// Setup builds the diagnostic logger and installs it as the slog default,
// which also routes the standard log package through it. With an empty path,
// or a path that cannot be opened, every record is discarded. The returned
// closer releases the log file.
func Setup(fs afero.Fs, path string) (*slog.Logger, io.Closer) {
	logger, closer := newLogger(fs, path)
	slog.SetDefault(logger)

	return logger, closer
}

// Discard returns a logger that drops every record.
func Discard() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func newLogger(fs afero.Fs, path string) (*slog.Logger, io.Closer) {
	if path == "" {
		return Discard(), nopCloser{}
	}

	file, err := fs.OpenFile(filepath.Clean(path), logFileFlags, logFilePerm)
	if err != nil {
		return Discard(), nopCloser{}
	}

	handler := slog.NewTextHandler(file, &slog.HandlerOptions{Level: slog.LevelDebug})

	return slog.New(handler), file
}

type nopCloser struct{}

func (nopCloser) Close() error { return nil }
