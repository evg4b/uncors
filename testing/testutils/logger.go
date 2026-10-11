package testutils

import (
	"bytes"
	"log/slog"
)

func NewBufferLogger() (*slog.Logger, *bytes.Buffer) {
	buffer := &bytes.Buffer{}
	handler := slog.NewTextHandler(buffer, &slog.HandlerOptions{Level: slog.LevelDebug})

	return slog.New(handler), buffer
}
