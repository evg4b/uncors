package uncorsapp

import (
	"log"
	"strings"
	"sync"
)

const (
	historyInitialCapacity = 1024

	// historyMaxLines bounds the scrollback. A long-running proxy logs a line
	// per request, so an unbounded buffer grows for as long as the process
	// lives; the oldest lines are the ones a user is least likely to want.
	historyMaxLines = 10_000
)

// history stores the most recent log lines in memory, discarding the oldest
// once historyMaxLines is reached.
type history struct {
	mu    sync.RWMutex
	lines []string
}

func newHistory() *history {
	log.Println("Initializing new history")

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
	newLines := strings.Split(line, "\n")
	h.lines = append(h.lines, newLines...)

	if overflow := len(h.lines) - historyMaxLines; overflow > 0 {
		// Copy down rather than reslicing, so the backing array of the dropped
		// lines can actually be collected.
		h.lines = append(h.lines[:0], h.lines[overflow:]...)
	}

	log.Printf("Appended %d lines to history (total lines: %d)", len(newLines), len(h.lines))
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

	log.Printf("Closing history with %d lines", len(h.lines))
	h.lines = nil

	return nil
}
