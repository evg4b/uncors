package tui

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
	"github.com/evg4b/uncors/internal/tui/render"
	"github.com/evg4b/uncors/internal/tui/styles"
)

type (
	requestStartedMsg   events.RequestStarted
	requestCompletedMsg events.RequestCompleted
)

type TrackerWidget struct {
	pending   map[uint64]events.RequestStarted
	ticking   bool
	spinner   spinner.Model
	formatter *render.Formatter
}

func NewTrackerWidget() *TrackerWidget {
	telemetry.Debug(context.Background(), "Creating TrackerWidget")

	loader := spinner.New()
	loader.Spinner = spinner.Meter
	// Plain: render.Formatter styles the frame together with the method.
	loader.Style = lipgloss.NewStyle()

	return &TrackerWidget{
		pending:   make(map[uint64]events.RequestStarted),
		spinner:   loader,
		formatter: render.NewFormatter(styles.Default),
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
	}

	return m, nil
}

func (m *TrackerWidget) View() tea.View {
	return tea.NewView(strings.Join(m.Lines(), "\n"))
}

func (m *TrackerWidget) Lines() []string {
	if len(m.pending) == 0 {
		return nil
	}

	lines := make([]string, 0, len(m.pending))

	for _, id := range slices.Sorted(maps.Keys(m.pending)) {
		lines = append(lines, m.formatter.Pending(m.pending[id], m.spinner.View()))
	}

	return lines
}

func (m *TrackerWidget) startTicking() tea.Cmd {
	if len(m.pending) == 0 || m.ticking {
		return nil
	}

	telemetry.Debug(context.Background(), "TrackerWidget: starting tick")

	m.ticking = true

	return m.spinner.Tick
}
