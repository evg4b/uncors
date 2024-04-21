package tui

import (
	"fmt"

	"github.com/evg4b/uncors/internal/tui/styles"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const unLetters = `██    ██ ███    ██
██    ██ ████   ██ 
██    ██ ██ ██  ██ 
██    ██ ██  ██ ██ 
 ██████  ██   ████`

const corsLetters = ` ██████  ██████  ██████  ███████ 
██      ██    ██ ██   ██ ██      
██      ██    ██ ██████  ███████ 
██      ██    ██ ██   ██      ██
 ██████  ██████  ██   ██ ███████`

var PrintLogoCmd = func(version string) tea.Cmd {
	return tea.Println(lipgloss.JoinVertical(
		lipgloss.Right,
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			styles.LogoRed.Render(unLetters),
			styles.LogoYellow.Render(corsLetters),
		),
		fmt.Sprintf("version: %s", version),
	))
}
