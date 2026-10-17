package uncorsapp

import (
	"context"
	"log/slog"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/evg4b/uncors/internal/telemetry"
)

type outputLineMsg string

type HistoryWidget struct {
	hist       *history
	keys       KeyMap
	autoScroll bool
	termWidth  int
	height     int
	offset     int
}

func NewHistoryWidget(keys KeyMap) *HistoryWidget {
	telemetry.Debug(context.Background(), "Creating HistoryWidget")

	hist := newHistory()

	return &HistoryWidget{
		hist:       hist,
		keys:       keys,
		autoScroll: true,
	}
}

func (m *HistoryWidget) Init() tea.Cmd {
	return nil
}

func (m *HistoryWidget) Update(msg tea.Msg) (*HistoryWidget, tea.Cmd) {
	switch typedMsg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termWidth = typedMsg.Width

		if m.autoScroll {
			m.gotoBottom()
		}

	case outputLineMsg:
		m.AppendLine(string(typedMsg))

	case tea.KeyPressMsg:
		m.handleKeyPress(typedMsg)
	}

	return m, nil
}

// AppendLine adds output to the history and keeps it scrolled to the bottom
// unless the user scrolled up.
func (m *HistoryWidget) AppendLine(line string) {
	evicted := m.hist.AppendLine(line)

	if m.autoScroll {
		m.gotoBottom()
	} else {
		m.scrollTo(m.offset - evicted)
	}
}

func (m *HistoryWidget) SetHeight(height int) {
	if telemetry.Enabled() {
		telemetry.Debug(context.Background(), "HistoryWidget: setting height", slog.Int("height", height))
	}

	m.height = height
}

func (m *HistoryWidget) HasLines() bool {
	return m.hist.LineCount() > 0
}

func (m *HistoryWidget) Close() error {
	telemetry.Debug(context.Background(), "HistoryWidget: closing")

	return m.hist.Close()
}

func (m *HistoryWidget) View() tea.View {
	if m.termWidth <= 0 || m.height <= 0 {
		return tea.NewView("")
	}

	lines := m.hist.Window(m.offset, m.offset+m.height)
	for index, line := range lines {
		if ansi.StringWidth(line) > m.termWidth {
			lines[index] = ansi.Cut(line, 0, m.termWidth)
		}
	}

	content := lipgloss.NewStyle().
		Width(m.termWidth).
		Height(m.height).
		Render(strings.Join(lines, "\n"))

	return tea.NewView(content)
}

func (m *HistoryWidget) maxOffset() int {
	return max(0, m.hist.LineCount()-m.height)
}

func (m *HistoryWidget) scrollTo(offset int) {
	m.offset = max(0, min(offset, m.maxOffset()))
}

func (m *HistoryWidget) gotoBottom() {
	m.offset = m.maxOffset()
}

func (m *HistoryWidget) atBottom() bool {
	return m.offset >= m.maxOffset()
}

func (m *HistoryWidget) handleKeyPress(msg tea.KeyPressMsg) {
	switch {
	case key.Matches(msg, m.keys.ScrollUp):
		m.scrollTo(m.offset - 1)
		m.autoScroll = m.atBottom()
	case key.Matches(msg, m.keys.ScrollDown):
		m.scrollTo(m.offset + 1)
		m.autoScroll = m.atBottom()
	case key.Matches(msg, m.keys.PageUp):
		m.scrollTo(m.offset - m.height)
		m.autoScroll = m.atBottom()
	case key.Matches(msg, m.keys.PageDown):
		m.scrollTo(m.offset + m.height)
		m.autoScroll = m.atBottom()
	case key.Matches(msg, m.keys.GotoTop):
		m.offset = 0
		m.autoScroll = false
	case key.Matches(msg, m.keys.GotoBottom):
		m.gotoBottom()
		m.autoScroll = true
	}
}
