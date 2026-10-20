package main

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/evg4b/uncors/testing/stories"
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF79C6"))
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#8BE9FD"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#777777"))
)

type picker struct {
	names   []string
	filter  string
	visible []string
	cursor  int
	chosen  string
	done    bool
}

func newPicker(catalog []stories.Story, last string) *picker {
	names := make([]string, 0, len(catalog))
	for _, story := range catalog {
		names = append(names, story.Name)
	}

	chooser := &picker{names: names}
	chooser.applyFilter()
	chooser.cursor = max(findStory(catalog, last), 0)

	return chooser
}

func pick(catalog []stories.Story, last string) (string, error) {
	chooser := newPicker(catalog, last)

	_, err := tea.NewProgram(chooser).Run()

	return chooser.chosen, err
}

func (p *picker) Init() tea.Cmd {
	return nil
}

func (p *picker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return p, nil
	}

	switch keyMsg.String() {
	case "ctrl+c", "esc":
		return p, p.finish("")
	case "enter":
		if len(p.visible) == 0 {
			return p, p.finish("")
		}

		return p, p.finish(p.visible[p.cursor])
	}

	p.edit(keyMsg)

	return p, nil
}

func (p *picker) View() tea.View {
	if p.done {
		return tea.NewView("")
	}

	lines := []string{titleStyle.Render("uncors stories") + "  " + p.filter + "▏", ""}

	for i, name := range p.visible {
		if i == p.cursor {
			lines = append(lines, selectedStyle.Render("› "+name))
		} else {
			lines = append(lines, "  "+name)
		}
	}

	if len(p.visible) == 0 {
		lines = append(lines, dimStyle.Render("  no stories match"))
	}

	lines = append(lines, "", dimStyle.Render("type to filter · ↑/↓ move · enter show · esc cancel"))

	return tea.NewView(strings.Join(lines, "\n"))
}

func (p *picker) finish(chosen string) tea.Cmd {
	p.chosen = chosen
	p.done = true

	return tea.Quit
}

func (p *picker) edit(msg tea.KeyPressMsg) {
	switch msg.String() {
	case "up", "ctrl+p":
		p.cursor = max(p.cursor-1, 0)
	case "down", "ctrl+n":
		p.cursor = min(p.cursor+1, max(len(p.visible)-1, 0))
	case "backspace":
		if p.filter != "" {
			p.filter = p.filter[:len(p.filter)-1]
			p.applyFilter()
		}
	default:
		if msg.Text != "" {
			p.filter += msg.Text
			p.applyFilter()
		}
	}
}

func (p *picker) applyFilter() {
	p.visible = p.visible[:0]

	for _, name := range p.names {
		if strings.Contains(name, p.filter) {
			p.visible = append(p.visible, name)
		}
	}

	p.cursor = 0
}
