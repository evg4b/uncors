package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/tui/styles"
)

func PrintMappings(mappings config.Mappings) tea.Cmd {
	return tea.Println(lipgloss.JoinHorizontal(
		lipgloss.Top,
		styles.InfoBlock.Render("INFO"),
		mappings.String(),
	))
}
