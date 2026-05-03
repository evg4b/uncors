package uncorsapp

import (
	"fmt"
	"log"
	"strings"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
	"charm.land/bubbles/v2/spinner"
)

const prefixWidth = 13

var pendingMethodStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#FFFFFF")).
	Background(lipgloss.Color("#8C8C8C")).
	Bold(true)

var pendingTextStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#8C8C8C"))

type (
	requestEventMsg requestEvent
)

type TrackerWidget struct {
	pending map[uint64]requestEvent
	ticking bool
	spinner spinner.Model
}

func NewTrackerWidget() *TrackerWidget {
	log.Println("Creating TrackerWidget")

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true)

	return &TrackerWidget{
		pending: make(map[uint64]requestEvent),
		spinner: s,
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
			existing, ok := m.pending[typedMsg.id]
			if ok {
				if typedMsg.prefix != "" {
					existing.prefix = typedMsg.prefix
				}
				m.pending[typedMsg.id] = existing
			} else {
				log.Printf("TrackerWidget: request started: %d %s %s", typedMsg.id, typedMsg.method, typedMsg.url.String())
				m.pending[typedMsg.id] = requestEvent(typedMsg)
			}
		}

		var cmd tea.Cmd

		if len(m.pending) > 0 && !m.ticking {
			log.Println("TrackerWidget: starting tick")

			m.ticking = true
			cmd = m.tickCmd()
		}

		return m, cmd

	case spinner.TickMsg:
		if len(m.pending) > 0 {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)

			return m, cmd
		}

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
		content := fmt.Sprintf("%s %s", m.spinner.View(), req.method)
		prefix := pendingMethodStyle.Width(prefixWidth).Render(content)
		url := pendingTextStyle.Render(req.url.String())

		if len(req.prefix) > 0 {
			fmt.Fprintf(&viewBuilder, "%s %s %s\n",
				req.prefix,
				prefix,
				url,
			)
		} else {
			fmt.Fprintf(&viewBuilder, "%s %s\n",
				prefix,
				url,
			)
		}
	}

	// Trim trailing newline for cleaner composition
	res := strings.TrimSuffix(viewBuilder.String(), "\n")

	return tea.NewView(res)
}

func (m *TrackerWidget) tickCmd() tea.Cmd {
	return m.spinner.Tick
}

