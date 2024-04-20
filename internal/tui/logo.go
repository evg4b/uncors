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
 ██████  ██   ████`

const corsLetters = ` ██████  ██████  ██████  ███████
██      ██    ██ ██   ██ ██     
██      ██    ██ ██████  ███████
██      ██    ██ ██   ██      ██
 ██████  ██████  ██   ██ ███████`

func Logo(version string) string {
	return lipgloss.JoinVertical(
		lipgloss.Right,
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			styles.LogoRed.Render(unLetters),
			styles.LogoYellow.Render(corsLetters),
		),
		fmt.Sprintf("version: %s", version),
	)
}
