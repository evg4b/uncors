package tui

import (
	"context"
	"log/slog"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/evg4b/uncors/internal/telemetry"
)

type outputLineMsg string

type HistoryWidget struct {
	hist       *history
	keys       KeyMap
	autoScroll bool
	termWidth  int
	blankLine  string
	height     int
	offset     int
	tail       []string
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
		m.blankLine = strings.Repeat(" ", max(0, typedMsg.Width))

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

func (m *HistoryWidget) AppendLine(line string) {
	evicted := m.hist.AppendLine(line)

	if m.autoScroll {
		m.gotoBottom()
	} else {
		m.scrollTo(m.offset - evicted)
	}
}

func (m *HistoryWidget) SetTail(lines []string) {
	m.tail = lines

	if m.autoScroll {
		m.gotoBottom()
	} else {
		m.scrollTo(m.offset)
	}
}

func (m *HistoryWidget) SetHeight(height int) {
	if telemetry.Enabled() {
		telemetry.Debug(context.Background(), "HistoryWidget: setting height", slog.Int("height", height))
	}

	m.height = height
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
	tailStart := max(0, m.offset-m.hist.LineCount()) - len(lines)

	var builder strings.Builder

	builder.Grow(m.height * (m.termWidth + 1))

	for index := range m.height {
		if index > 0 {
			builder.WriteByte('\n')
		}

		var line string

		switch {
		case index < len(lines):
			line = lines[index]
		case tailStart+index < len(m.tail):
			line = m.tail[tailStart+index]
		default:
			builder.WriteString(m.blankLine)

			continue
		}

		width := ansi.StringWidth(line)
		if width > m.termWidth {
			line = ansi.Cut(line, 0, m.termWidth)
			width = m.termWidth
		}

		builder.WriteString(line)
		builder.WriteString(m.blankLine[:m.termWidth-width])
	}

	return tea.NewView(builder.String())
}

func (m *HistoryWidget) maxOffset() int {
	return max(0, m.hist.LineCount()+len(m.tail)-m.height)
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
