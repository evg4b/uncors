package tui

import (
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errInternalWrite = fmt.Errorf("internal write error")

type errorWriterInternal struct{}

func (e *errorWriterInternal) Write(_ []byte) (int, error) {
	return 0, errInternalWrite
}

func TestFlushBuffer_EmptyBuffer(t *testing.T) {
	out := NewCliOutput(io.Discard)
	err := out.flushBuffer()
	require.NoError(t, err)
}

func TestPrintMessageBox_PanicsOnWriteError(t *testing.T) {
	assert.Panics(t, func() {
		printMessageBox(&errorWriterInternal{}, "test message", "INFO", levelStyles[infoOutput])
	})
}
