package uncorsapp

import (
	"fmt"
	"log"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
)

var pendingMethodStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#FFFFFF")).
	Background(lipgloss.Color("#8C8C8C")).
	Padding(0, 1)

type (
	requestEventMsg requestEvent
	tickMsg         struct{}
)

type TrackerWidget struct {
	pending map[uint64]requestEvent
	ticking bool
}

func NewTrackerWidget() *TrackerWidget {
	log.Println("Creating TrackerWidget")

	return &TrackerWidget{
		pending: make(map[uint64]requestEvent),
	}
}

func (m *TrackerWidget) Init() tea.Cmd {
	return nil
}

func (m *TrackerWidget) Update(msg tea.Msg) (*TrackerWidget, tea.Cmd) {
	switch typedMsg := msg.(type) {
	case requestEventMsg:
		if typedMsg.done {
			log.Printf("TrackerWidget: request done: %d", typedMsg.id)
			delete(m.pending, typedMsg.id)
		} else {
			log.Printf("TrackerWidget: request started: %d %s %s", typedMsg.id, typedMsg.method, typedMsg.url.String())
			m.pending[typedMsg.id] = requestEvent(typedMsg)
		}

		var cmd tea.Cmd

		if len(m.pending) > 0 && !m.ticking {
			log.Println("TrackerWidget: starting tick")

			m.ticking = true
			cmd = m.tickCmd()
		}

		return m, cmd

	case tickMsg:
		if len(m.pending) > 0 {
			return m, m.tickCmd()
		}

		log.Println("TrackerWidget: stopping tick")

		m.ticking = false

		return m, nil

	case restartMsg:
		log.Println("TrackerWidget: handling restartMsg")

		m.pending = make(map[uint64]requestEvent)
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

	for _, req := range m.pending {
		elapsed := formatElapsed(time.Since(req.startedAt))
		fmt.Fprintf(&viewBuilder, "  %s %s  %s\n",
			pendingMethodStyle.Render(req.method),
			req.url.String(),
			elapsed,
		)
	}

	// Trim trailing newline for cleaner composition
	res := strings.TrimSuffix(viewBuilder.String(), "\n")

	return tea.NewView(res)
}

func (m *TrackerWidget) tickCmd() tea.Cmd {
	return tea.Tick(tickInterval, func(_ time.Time) tea.Msg {
		return tickMsg{}
	})
}

func formatElapsed(duration time.Duration) string {
	duration = duration.Truncate(time.Millisecond)
	if duration < time.Second {
		return fmt.Sprintf("%dms", duration.Milliseconds())
	}

	return fmt.Sprintf("%.1fs", duration.Seconds())
}
