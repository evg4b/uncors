package tui

import (
	"fmt"
	"net/url"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/evg4b/uncors/internal/events"
)

const (
	benchWidth   = 120
	benchHeight  = 40
	benchLines   = 200
	benchPending = 5
)

func BenchmarkHistoryView(b *testing.B) {
	history := NewHistoryWidget(NewKeyMap())

	b.Cleanup(func() { _ = history.Close() })

	for index := range benchLines {
		history.AppendLine(fmt.Sprintf("\x1b[48;2;84;90;201m PROXY \x1b[m 200 GET https://api.local/users/%d", index))
	}

	history.Update(tea.WindowSizeMsg{Width: benchWidth, Height: benchHeight})
	history.SetHeight(benchHeight)

	b.ReportAllocs()

	for b.Loop() {
		history.View()
	}
}

func BenchmarkTrackerView(b *testing.B) {
	tracker := NewTrackerWidget()
	reqURL, _ := url.Parse("https://api.local/users")

	for id := range uint64(benchPending) {
		tracker.pending[id] = events.RequestStarted{ID: id, Method: "GET", URL: reqURL, Handler: events.HandlerProxy}
	}

	b.ReportAllocs()

	for b.Loop() {
		tracker.View()
	}
}

func BenchmarkMemoryView(b *testing.B) {
	memory := NewMemoryWidget(WithMemoryUsage(func() float64 { return 42.5 }))

	b.ReportAllocs()

	for b.Loop() {
		memory.View()
	}
}
