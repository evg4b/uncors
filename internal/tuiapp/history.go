package tuiapp

import (
	"context"
	"log/slog"
	"strings"
	"sync"

	"github.com/evg4b/uncors/internal/telemetry"
)

const (
	historyCapacity = 10_000
)

type history struct {
	mu    sync.RWMutex
	lines []string
	start int
}

func newHistory() *history {
	telemetry.Debug(context.Background(), "Initializing new history")

	return &history{lines: make([]string, 0, historyCapacity)}
}

func (h *history) AppendLine(line string) int {
	h.mu.Lock()
	defer h.mu.Unlock()

	line = strings.TrimRight(line, "\n")
	newLines := strings.Split(line, "\n")
	evicted := 0

	for _, newLine := range newLines {
		if len(h.lines) < historyCapacity {
			h.lines = append(h.lines, newLine)

			continue
		}

		h.lines[h.start] = newLine
		h.start = (h.start + 1) % historyCapacity
		evicted++
	}

	if telemetry.Enabled() {
		telemetry.Debug(context.Background(), "appended lines to history",
			slog.Int("lines", len(newLines)), slog.Int("total", len(h.lines)))
	}

	return evicted
}

func (h *history) Window(from, to int) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	from = max(from, 0)
	to = min(to, len(h.lines))

	if from >= to {
		return nil
	}

	window := make([]string, 0, to-from)
	for index := from; index < to; index++ {
		window = append(window, h.lines[(h.start+index)%len(h.lines)])
	}

	return window
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
	h.start = 0

	return nil
}
