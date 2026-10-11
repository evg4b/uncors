package uncorsapp

import (
	"log/slog"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/tui/styles"
	"github.com/evg4b/uncors/pkg/urlt"
)

var pendingMethodStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#FFFFFF")).
	Background(lipgloss.Color("#8C8C8C")).
	PaddingLeft(1).
	PaddingRight(1).
	Bold(true)

var pendingTextStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#8C8C8C"))

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
	slog.Debug("Creating TrackerWidget")

	loader := spinner.New()
	loader.Spinner = spinner.Meter
	loader.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true)

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
		slog.Debug("TrackerWidget: request started", "id", typedMsg.ID, "method", typedMsg.Method)
		m.pending[typedMsg.ID] = events.RequestStarted(typedMsg)

		return m, m.startTicking()

	case requestCompletedMsg:
		slog.Debug("TrackerWidget: request done", "id", typedMsg.ID)
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

	case restartMsg:
		slog.Debug("TrackerWidget: handling restartMsg")

		m.pending = make(map[uint64]events.RequestStarted)
		m.ticking = false

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

	return 1 + len(m.pending) // "In progress:" header + N request lines
}

func (m *TrackerWidget) View() tea.View {
	if len(m.pending) == 0 {
		return tea.NewView("")
	}

	var viewBuilder strings.Builder

	index := 0

	for _, req := range m.pending {
		urlStr := ""
		if req.URL != nil {
			urlStr = urlt.URL_String(req.URL)
		}

		url := pendingTextStyle.Render(urlStr)

		if label := styles.HandlerLabel(req.Handler); label != "" {
			viewBuilder.WriteString(label)
		}

		viewBuilder.WriteString(pendingMethodStyle.Render(m.spinner.View()))
		viewBuilder.WriteString(pendingMethodStyle.Render(req.Method))
		viewBuilder.WriteRune(' ')
		viewBuilder.WriteString(url)

		if req.Rewrite != nil {
			viewBuilder.WriteString(" " + styles.DimTextStyle.Render("→ "+req.Rewrite.To))
		}

		if index != 0 {
			viewBuilder.WriteRune('\n')
		}

		index++
	}

	return tea.NewView(viewBuilder.String())
}

// startTicking starts the spinner unless it is already running.
func (m *TrackerWidget) startTicking() tea.Cmd {
	if len(m.pending) == 0 || m.ticking {
		return nil
	}

	slog.Debug("TrackerWidget: starting tick")

	m.ticking = true

	return m.spinner.Tick
}
