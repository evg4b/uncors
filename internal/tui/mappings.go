package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/evg4b/uncors/internal/config"
)

func PrintMappings(mappings config.Mappings) tea.Cmd {
	return tea.Println(mappings.String())
}
