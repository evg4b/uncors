package log

import (
	"log"

	"github.com/pterm/pterm"
)

func StandardErrorLogAdapter() *log.Logger {
	return log.New(&writerAdapter{printer: &errorPrinter}, "", 0)
}

type writerAdapter struct {
	printer *pterm.PrefixPrinter
}

func (w *writerAdapter) Write(p []byte) (int, error) {
	w.printer.Print(string(p))

	return len(p), nil
}
