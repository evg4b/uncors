package uncorsapp

import (
	"strings"
	"sync"
)

const (
	historyInitialCapacity = 1024
)

// history stores log lines in memory without a fixed limit.
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
// Multi-line strings are split on '\n' so the viewport receives one entry per visual row.
func (h *history) AppendLine(line string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	line = strings.TrimRight(line, "\n")
	h.lines = append(h.lines, strings.Split(line, "\n")...)
}

// Lines returns a copy of the slice of all stored lines.
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
