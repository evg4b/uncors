package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evg4b/uncors/internal/tui/styles"
)

var PrintDisclaimerMessage = func() tea.Cmd {
	height := lipgloss.Height(DisclaimerMessage)
	space := strings.Repeat("\n", height-1)

	return tea.Println(lipgloss.JoinHorizontal(
		lipgloss.Top,
		styles.WarningBlock.Render(strings.ToUpper(styles.WarningLabel), space),
		styles.WarningText.Render(DisclaimerMessage),
	))
}
