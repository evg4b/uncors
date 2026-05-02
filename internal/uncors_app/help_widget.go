package uncorsapp

import (
	help "charm.land/bubbles/v2/help"
	key "charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
)

type HelpWidget struct {
	help help.Model
	keys keyMap
}

func NewHelpWidget(keys keyMap) *HelpWidget {
	return &HelpWidget{
		help: help.New(),
		keys: keys,
	}
}

func (m *HelpWidget) Init() tea.Cmd {
	return nil
}

func (m *HelpWidget) Update(msg tea.Msg) (*HelpWidget, tea.Cmd) {
	switch typedMsg := msg.(type) {
	case tea.WindowSizeMsg:
		m.help.SetWidth(typedMsg.Width)
	case tea.KeyPressMsg:
		if key.Matches(typedMsg, m.keys.Help) {
			m.help.ShowAll = !m.help.ShowAll
		}
	}

	return m, nil
}

func (m *HelpWidget) Height() int {
	return lipgloss.Height(m.help.View(m.keys))
}

func (m *HelpWidget) View() tea.View {
	return tea.NewView(m.help.View(m.keys))
}
