package uncorsapp

import (
	help "charm.land/bubbles/v2/help"
	key "charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/evg4b/uncors/internal/tui"
)

type uncorsApp struct {
	version string
	keys    keyMap
	help    help.Model
}

func NewModles(version string) tea.Model {
	return uncorsApp{
		version: version,
		keys:    newKeyMap(),
		help:    help.New(),
	}
}

func (m uncorsApp) Init() tea.Cmd {
	return tea.Sequence(
		tea.ClearScreen,
		tea.Println(
			tui.Logo(m.version),
		),
	)
}

func (m uncorsApp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.help.SetWidth(msg.Width)
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m uncorsApp) View() tea.View {
	return tea.NewView(
		m.help.View(m.keys),
	)
}
