package render_test

import (
	"io"
	"net/http"
	"testing"

	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/tui/render"
	"github.com/evg4b/uncors/internal/tui/styles"
)

func BenchmarkFormatRequest(b *testing.B) {
	event := events.RequestCompleted{
		Handler: events.HandlerProxy,
		Data:    makeRequestData(http.MethodGet, "https://api.domain.com/api/user", http.StatusOK),
	}

	formatter := render.NewFormatter(styles.Default)

	b.ReportAllocs()

	for b.Loop() {
		formatter.Request(event)
	}
}

func BenchmarkFormatPending(b *testing.B) {
	event := events.RequestStarted{
		Handler: events.HandlerProxy,
		Method:  http.MethodGet,
		URL:     makeRequestData(http.MethodGet, "https://api.domain.com/api/user", 0).URL,
	}

	formatter := render.NewFormatter(styles.Default)

	b.ReportAllocs()

	for b.Loop() {
		formatter.Pending(event, "▱▱▱")
	}
}

func BenchmarkOutputInfo(b *testing.B) {
	output := render.NewCliOutput(io.Discard)

	b.ReportAllocs()

	for b.Loop() {
		output.Info("Recording HAR to ./traffic.har")
	}
}
