package mocks

import (
	"io"
	"testing"

	"github.com/charmbracelet/log"
)

func NewNoopLogger(t *testing.T) *log.Logger {
	t.Helper()

	return log.New(io.Discard)
}
