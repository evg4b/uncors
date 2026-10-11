package logging

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
)

const EnvVar = "UNCORS_LOGGING"

const (
	logFileFlags = os.O_CREATE | os.O_WRONLY | os.O_APPEND
	logFilePerm  = 0o644
)

func Setup(fs afero.Fs, path string) (*slog.Logger, io.Closer) {
	logger, closer := newLogger(fs, path)
	slog.SetDefault(logger)

	return logger, closer
}

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
