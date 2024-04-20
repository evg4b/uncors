package log

import (
	"io"

	"github.com/pterm/pterm"
)

func SetOutput(output io.Writer) {
	pterm.SetDefaultOutput(output)
}
