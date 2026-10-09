package uncorsapp

import (
	"strings"
)

// channelWriter turns rendered console output into history lines.
//
// CliOutput writes one complete line per call, so this is simply where the
// TUI's console output goes instead of a terminal. Nothing renders here: the
// service hands the model structured events, internal/render decides what they
// say, and CliOutput decides how they look.
type channelWriter struct {
	ch chan<- string
}

func newChannelWriter(ch chan<- string) *channelWriter {
	return &channelWriter{ch: ch}
}

// Write never blocks. A model that cannot keep up drops lines rather than
// stalling whatever produced them, which is the same policy the request
// tracker applies to activity.
func (w *channelWriter) Write(line []byte) (int, error) {
	msg := strings.TrimRight(string(line), "\n")
	if len(msg) > 0 {
		select {
		case w.ch <- msg:
		default:
		}
	}

	return len(line), nil
}
