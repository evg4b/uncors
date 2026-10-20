package uncorsapp

import (
	"context"
	"log/slog"
	"maps"
	"slices"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/telemetry"
	"github.com/evg4b/uncors/internal/tui"
)

type (
	requestStartedMsg   events.RequestStarted
	requestCompletedMsg events.RequestCompleted
)

type TrackerWidget struct {
	pending map[uint64]events.RequestStarted
	ticking bool
	spinner spinner.Model
}

func NewTrackerWidget() *TrackerWidget {
	telemetry.Debug(context.Background(), "Creating TrackerWidget")

	loader := spinner.New()
	loader.Spinner = spinner.Meter
	// Plain: tui.FormatPending styles the frame together with the method.
	loader.Style = lipgloss.NewStyle()

	return &TrackerWidget{
		pending: make(map[uint64]events.RequestStarted),
		spinner: loader,
	}
}

func (m *TrackerWidget) Init() tea.Cmd {
	return nil
}

func (m *TrackerWidget) Update(msg tea.Msg) (*TrackerWidget, tea.Cmd) {
	switch typedMsg := msg.(type) {
	case requestStartedMsg:
		if telemetry.Enabled() {
			telemetry.Debug(context.Background(), "TrackerWidget: request started",
				slog.Uint64("id", typedMsg.ID), slog.String("method", typedMsg.Method))
		}

		m.pending[typedMsg.ID] = events.RequestStarted(typedMsg)

		return m, m.startTicking()

	case requestCompletedMsg:
		if telemetry.Enabled() {
			telemetry.Debug(context.Background(), "TrackerWidget: request done", slog.Uint64("id", typedMsg.ID))
		}

		delete(m.pending, typedMsg.ID)

		return m, nil
	case spinner.TickMsg:
		if len(m.pending) > 0 {
			var cmd tea.Cmd

			m.spinner, cmd = m.spinner.Update(msg)

			return m, cmd
		}

		m.ticking = false

		return m, nil

	case lifecycleMsg:
		// Requests of the previous generation finished before the restart, so
		// anything still pending lost its completion event.
		if _, restarted := typedMsg.event.(events.ServerRestarted); restarted {
			telemetry.Debug(context.Background(), "TrackerWidget: clearing pending requests after restart")

			m.pending = make(map[uint64]events.RequestStarted)
			m.ticking = false
		}

		return m, nil
	}

	return m, nil
}

func (m *TrackerWidget) ActiveCount() int {
	return len(m.pending)
}

func (m *TrackerWidget) Height() int {
	if len(m.pending) == 0 {
		return 0
	}

	return 1 + len(m.pending)
}

func (m *TrackerWidget) View() tea.View {
	if len(m.pending) == 0 {
		return tea.NewView("")
	}

	lines := make([]string, 0, len(m.pending))

	for _, id := range slices.Sorted(maps.Keys(m.pending)) {
		lines = append(lines, tui.FormatPending(m.pending[id], m.spinner.View()))
	}

	return tea.NewView(strings.Join(lines, "\n"))
}

func (m *TrackerWidget) startTicking() tea.Cmd {
	if len(m.pending) == 0 || m.ticking {
		return nil
	}

	telemetry.Debug(context.Background(), "TrackerWidget: starting tick")

	m.ticking = true

	return m.spinner.Tick
}
