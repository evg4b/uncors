package uncorsapp

import (
	"context"
	"log/slog"
	"strings"
	"sync"

	"github.com/evg4b/uncors/internal/telemetry"
)

const (
	historyInitialCapacity = 1024
)

type history struct {
	mu    sync.RWMutex
	lines []string
}

func newHistory() *history {
	telemetry.Debug(context.Background(), "Initializing new history")

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

	if telemetry.Enabled() {
		telemetry.Debug(context.Background(), "appended lines to history",
			slog.Int("lines", len(newLines)), slog.Int("total", len(h.lines)))
	}
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

	telemetry.Debug(context.Background(), "closing history", slog.Int("lines", len(h.lines)))
	h.lines = nil

	return nil
}
