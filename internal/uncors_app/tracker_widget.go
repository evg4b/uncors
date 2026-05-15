package uncorsapp

import (
	"log"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/evg4b/uncors/internal/server"
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
	requestEventMsg server.RequestEvent
)

type TrackerWidget struct {
	pending map[uint64]server.RequestEvent
	ticking bool
	spinner spinner.Model
}

func NewTrackerWidget() *TrackerWidget {
	log.Println("Creating TrackerWidget")

	loader := spinner.New()
	loader.Spinner = spinner.Meter
	loader.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true)

	return &TrackerWidget{
		pending: make(map[uint64]server.RequestEvent),
		spinner: loader,
	}
}

func (m *TrackerWidget) Init() tea.Cmd {
	return nil
}

func (m *TrackerWidget) Update(msg tea.Msg) (*TrackerWidget, tea.Cmd) {
	switch typedMsg := msg.(type) {
	case requestEventMsg:
		return m.requestEventMsg(typedMsg)
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

		m.pending = make(map[uint64]server.RequestEvent)
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
			urlStr = req.URL.String()
		}

		url := pendingTextStyle.Render(urlStr)

		if len(req.Prefix) > 0 {
			viewBuilder.WriteString(req.Prefix)
		}

		viewBuilder.WriteString(pendingMethodStyle.Render(m.spinner.View()))
		viewBuilder.WriteString(pendingMethodStyle.Render(req.Method))
		viewBuilder.WriteRune(' ')
		viewBuilder.WriteString(url)

		if index != 0 {
			viewBuilder.WriteRune('\n')
		}

		index++
	}

	return tea.NewView(viewBuilder.String())
}

func (m *TrackerWidget) requestEventMsg(msg requestEventMsg) (*TrackerWidget, tea.Cmd) {
	if msg.Done {
		log.Printf("TrackerWidget: request done: %d", msg.ID)
		delete(m.pending, msg.ID)
	} else if msg.URL != nil {
		log.Printf("TrackerWidget: request started: %d %s %s", msg.ID, msg.Method, msg.URL.String())
		m.pending[msg.ID] = server.RequestEvent(msg)
	} else if event, ok := m.pending[msg.ID]; ok {
		event.Prefix = msg.Prefix
		m.pending[msg.ID] = event
	}

	var cmd tea.Cmd

	if len(m.pending) > 0 && !m.ticking {
		log.Println("TrackerWidget: starting tick")

		m.ticking = true
		cmd = m.spinner.Tick
	}

	return m, cmd
}
