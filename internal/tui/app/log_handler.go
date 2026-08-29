package app

import (
	"context"
	"log/slog"
	"strings"
	"sync"
)

// logHandlerBuffer bounds how many records are kept before the model starts
// draining them.
const logHandlerBuffer = 256

// LogHandler collects diagnostics for the terminal UI. Writing them to stderr
// would corrupt the alt-screen, so records are queued and rendered in the
// history view like any other line.
type LogHandler struct {
	level slog.Level

	mu      sync.Mutex
	records chan string
	attrs   []slog.Attr
	group   string
}

func NewLogHandler(level slog.Level) *LogHandler {
	return &LogHandler{
		level:   level,
		records: make(chan string, logHandlerBuffer),
	}
}

// Records is the stream of formatted diagnostics for the model to render.
func (h *LogHandler) Records() <-chan string {
	return h.records
}

func (h *LogHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *LogHandler) Handle(_ context.Context, record slog.Record) error {
	var builder strings.Builder

	builder.WriteString(record.Level.String())
	builder.WriteString(": ")
	builder.WriteString(record.Message)

	h.mu.Lock()
	attrs := h.attrs
	group := h.group
	h.mu.Unlock()

	for _, attr := range attrs {
		writeAttr(&builder, group, attr)
	}

	record.Attrs(func(attr slog.Attr) bool {
		writeAttr(&builder, group, attr)

		return true
	})

	select {
	case h.records <- builder.String():
	default:
		// The view is a convenience, never a reason to slow the proxy down.
	}

	return nil
}

func (h *LogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h.mu.Lock()
	defer h.mu.Unlock()

	return &LogHandler{
		level:   h.level,
		records: h.records,
		attrs:   append(append([]slog.Attr{}, h.attrs...), attrs...),
		group:   h.group,
	}
}

func (h *LogHandler) WithGroup(name string) slog.Handler {
	h.mu.Lock()
	defer h.mu.Unlock()

	return &LogHandler{
		level:   h.level,
		records: h.records,
		attrs:   append([]slog.Attr{}, h.attrs...),
		group:   name,
	}
}

func writeAttr(builder *strings.Builder, group string, attr slog.Attr) {
	builder.WriteByte(' ')

	if group != "" {
		builder.WriteString(group)
		builder.WriteByte('.')
	}

	builder.WriteString(attr.Key)
	builder.WriteByte('=')
	builder.WriteString(attr.Value.String())
}
