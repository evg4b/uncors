package app

import (
	"fmt"
	"log/slog"
)

// logLineMsg is a diagnostic record on its way to the history view.
type logLineMsg string

// logRecords returns the stream of diagnostics when the process logger is the
// TUI's own handler, and nil when diagnostics go somewhere else (a log file, or
// a test).
func logRecords() <-chan string {
	handler, ok := slog.Default().Handler().(*LogHandler)
	if !ok {
		return nil
	}

	return handler.Records()
}

// slogDebugf traces the model's own behaviour. These lines describe the widget
// plumbing rather than anything the user did, so they are debug level.
func slogDebugf(format string, args ...any) {
	if !slog.Default().Enabled(nil, slog.LevelDebug) { //nolint:staticcheck // no request context here
		return
	}

	slog.Debug(fmt.Sprintf(format, args...))
}
