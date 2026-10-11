package testutils

import (
	"bytes"
	"log/slog"
)

// NewBufferLogger returns a logger that writes text records to the returned
// buffer, so tests can assert on what a component logged.
func NewBufferLogger() (*slog.Logger, *bytes.Buffer) {
	buffer := &bytes.Buffer{}
	handler := slog.NewTextHandler(buffer, &slog.HandlerOptions{Level: slog.LevelDebug})

	return slog.New(handler), buffer
}
