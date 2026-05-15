package tui

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

var errInternalWrite = fmt.Errorf("internal write error")

type errorWriterInternal struct{}

func (e *errorWriterInternal) Write(_ []byte) (int, error) {
	return 0, errInternalWrite
}

func TestPrintMessageBox_PanicsOnWriteError(t *testing.T) {
	assert.Panics(t, func() {
		printMessageBox(&errorWriterInternal{}, "test message", "INFO", levelStyles[infoOutput])
	})
}
