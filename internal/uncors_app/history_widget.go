package uncorsapp

import (
	"context"
	"log/slog"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/evg4b/uncors/internal/telemetry"
)

type outputLineMsg string

type HistoryWidget struct {
	hist       *history
	vp         viewport.Model
	keys       KeyMap
	autoScroll bool
	termWidth  int
}

func NewHistoryWidget(keys KeyMap) *HistoryWidget {
	telemetry.Debug(context.Background(), "Creating HistoryWidget")

	hist := newHistory()

	return &HistoryWidget{
		hist:       hist,
		vp:         viewport.New(),
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
		m.vp.SetWidth(typedMsg.Width)

		if m.autoScroll {
			m.vp.GotoBottom()
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
	m.hist.AppendLine(line)
	m.vp.SetContentLines(m.hist.Lines())

	if m.autoScroll {
		m.vp.GotoBottom()
	}
}

func (m *HistoryWidget) SetHeight(height int) {
	if telemetry.Enabled() {
		telemetry.Debug(context.Background(), "HistoryWidget: setting height", slog.Int("height", height))
	}

	m.vp.SetHeight(height)
}

func (m *HistoryWidget) HasLines() bool {
	return m.hist.LineCount() > 0
}

func (m *HistoryWidget) Close() error {
	telemetry.Debug(context.Background(), "HistoryWidget: closing")

	return m.hist.Close()
}

func (m *HistoryWidget) View() tea.View {
	return tea.NewView(m.vp.View())
}

func (m *HistoryWidget) handleKeyPress(msg tea.KeyPressMsg) {
	switch {
	case key.Matches(msg, m.keys.ScrollUp):
		m.vp.ScrollUp(1)
		m.autoScroll = m.vp.AtBottom()
	case key.Matches(msg, m.keys.ScrollDown):
		m.vp.ScrollDown(1)
		m.autoScroll = m.vp.AtBottom()
	case key.Matches(msg, m.keys.PageUp):
		m.vp.PageUp()
		m.autoScroll = m.vp.AtBottom()
	case key.Matches(msg, m.keys.PageDown):
		m.vp.PageDown()
		m.autoScroll = m.vp.AtBottom()
	case key.Matches(msg, m.keys.GotoTop):
		m.vp.GotoTop()
		m.autoScroll = false
	case key.Matches(msg, m.keys.GotoBottom):
		m.vp.GotoBottom()
		m.autoScroll = true
	}
}
