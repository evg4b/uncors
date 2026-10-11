package uncorsapp

import (
	"log/slog"
	"strings"
	"sync"
)

const (
	historyInitialCapacity = 1024
)

type history struct {
	mu    sync.RWMutex
	lines []string
}

func newHistory() *history {
	slog.Debug("Initializing new history")

	return &history{
		lines: make([]string, 0, historyInitialCapacity),
	}
}

func (h *history) AppendLine(line string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	line = strings.TrimRight(line, "\n")
	newLines := strings.Split(line, "\n")
	h.lines = append(h.lines, newLines...)

	slog.Debug("appended lines to history", "lines", len(newLines), "total", len(h.lines))
}

func (h *history) Lines() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	res := make([]string, len(h.lines))
	copy(res, h.lines)

	return res
}

func (h *history) LineCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.lines)
}

func (h *history) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	slog.Debug("closing history", "lines", len(h.lines))
	h.lines = nil

	return nil
}
