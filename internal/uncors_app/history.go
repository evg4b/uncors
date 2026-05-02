package uncorsapp

import (
	"strings"
	"sync"
)

const (
	historyMaxLines        = 10000
	historyInitialCapacity = 1024
)

// history stores log lines in memory up to historyMaxLines.
type history struct {
	mu    sync.RWMutex
	lines []string
}

func newHistory() *history {
	return &history{
		lines: make([]string, 0, historyInitialCapacity),
	}
}

// AppendLine writes line to the history.
// Multi-line strings (logo, box messages) are split on '\n' so the viewport
// receives one entry per visual row.
func (h *history) AppendLine(line string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	line = strings.TrimRight(line, "\n")
	for subline := range strings.SplitSeq(line, "\n") {
		h.lines = append(h.lines, subline)
	}

	if len(h.lines) > historyMaxLines {
		// Drop the oldest lines
		excess := len(h.lines) - historyMaxLines
		h.lines = h.lines[excess:]
	}
}

// Lines returns the cached slice of all stored lines.
// The slice is valid until the next AppendLine call, but since it's just a
// view for bubbletea viewport, it's fine. Bubbletea viewport doesn't modify it.
// To prevent data races during rendering, we return a copy.
func (h *history) Lines() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	res := make([]string, len(h.lines))
	copy(res, h.lines)

	return res
}

// LineCount returns the total number of stored lines.
func (h *history) LineCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.lines)
}

// Close cleans up the history.
func (h *history) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.lines = nil

	return nil
}
