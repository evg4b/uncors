package uncorsapp

import (
	key "charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
)

type outputLineMsg string

type HistoryWidget struct {
	hist       *history
	vp         viewport.Model
	keys       keyMap
	autoScroll bool
	termWidth  int
}

func NewHistoryWidget(keys keyMap) *HistoryWidget {
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
		// Height is handled by SetHeight() method explicitly since the main app coordinates it.
		if m.autoScroll {
			m.vp.GotoBottom()
		}

	case outputLineMsg:
		atBottom := m.autoScroll
		m.hist.AppendLine(string(typedMsg))
		m.vp.SetContentLines(m.hist.Lines())

		if atBottom {
			m.vp.GotoBottom()
		}

	case restartMsg:
		// Reset state if needed, but history probably stays or clears?
		// In previous implementation, handleRestart() didn't clear history, it just recalculated height.
		return m, nil

	case tea.KeyPressMsg:
		m.handleKeyPress(typedMsg)
	}

	return m, nil
}

func (m *HistoryWidget) SetHeight(height int) {
	m.vp.SetHeight(height)
}

func (m *HistoryWidget) HasLines() bool {
	return m.hist.LineCount() > 0
}

func (m *HistoryWidget) Close() error {
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
