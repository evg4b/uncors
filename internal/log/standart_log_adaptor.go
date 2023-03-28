package log

import (
	"github.com/pterm/pterm"

	"log"
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
