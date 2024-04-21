package tui

import (
	"fmt"

	"github.com/evg4b/uncors/internal/tui/styles"

	"github.com/charmbracelet/lipgloss"
)

const unLetters = `██    ██ ███    ██ 
██    ██ ████   ██ 
██    ██ ██ ██  ██ 
██    ██ ██  ██ ██ 
 ██████  ██   ████ `

const corsLetters = ` ██████  ██████  ██████  ███████
██      ██    ██ ██   ██ ██     
██      ██    ██ ██████  ███████
██      ██    ██ ██   ██      ██
 ██████  ██████  ██   ██ ███████`

var (
	red    = styles.LogoRed.Render
	yellow = styles.LogoYellow.Render
)

func Logo(version string) string {
	return lipgloss.JoinVertical(
		lipgloss.Right,
		lipgloss.JoinHorizontal(lipgloss.Top, red(unLetters), yellow(corsLetters)),
		fmt.Sprintf("version: %s", version),
	)
}
