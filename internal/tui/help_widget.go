package tui

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type HelpWidget struct {
	help     help.Model
	keys     KeyMap
	rendered string
	height   int
}

func NewHelpWidget(keys KeyMap) *HelpWidget {
	widget := &HelpWidget{
		help: help.New(),
		keys: keys,
	}
	widget.refresh()

	return widget
}

func (m *HelpWidget) Init() tea.Cmd {
	return nil
}

func (m *HelpWidget) Update(msg tea.Msg) (*HelpWidget, tea.Cmd) {
	switch typedMsg := msg.(type) {
	case tea.WindowSizeMsg:
		m.help.SetWidth(typedMsg.Width)
		m.refresh()
	case tea.KeyPressMsg:
		if key.Matches(typedMsg, m.keys.Help) {
			m.help.ShowAll = !m.help.ShowAll
			m.refresh()
		}
	}

	return m, nil
}

func (m *HelpWidget) Height() int {
	return m.height
}

func (m *HelpWidget) View() tea.View {
	return tea.NewView(m.rendered)
}

func (m *HelpWidget) refresh() {
	m.rendered = m.help.View(m.keys)
	m.height = lipgloss.Height(m.rendered)
}
