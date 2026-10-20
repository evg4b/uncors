package main

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/evg4b/uncors/testing/stories"
)

const statusRows = 2

type viewer struct {
	story  stories.Story
	model  tea.Model
	width  int
	height int
}

func newViewer(story stories.Story) *viewer {
	return &viewer{story: story, model: story.New()}
}

func show(story stories.Story) error {
	_, err := tea.NewProgram(newViewer(story)).Run()

	return err
}

func (v *viewer) Init() tea.Cmd {
	return v.model.Init()
}

func (v *viewer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		v.width, v.height = typed.Width, typed.Height
		msg = tea.WindowSizeMsg{Width: v.width, Height: v.storyHeight()}
	case tea.KeyPressMsg:
		if typed.String() == "q" || typed.String() == "ctrl+c" {
			return v, tea.Quit
		}
	}

	var cmd tea.Cmd

	v.model, cmd = v.model.Update(msg)

	return v, cmd
}

func (v *viewer) View() tea.View {
	content := v.model.View().Content

	height := v.story.Height
	if height == 0 {
		height = lipgloss.Height(content)
	}

	height = min(height, v.storyHeight())

	status := dimStyle.Render(v.story.Name + " · q quit")
	frame := stories.DrawFrame(content, max(v.width, 1), max(height, 1))

	view := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, frame, "", status))
	view.AltScreen = true

	return view
}

func (v *viewer) storyHeight() int {
	free := max(v.height-statusRows, 1)

	if v.story.Height > 0 {
		return min(v.story.Height, free)
	}

	return free
}
