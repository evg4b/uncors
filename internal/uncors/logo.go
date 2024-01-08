package uncors

import (
	"fmt"

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

var (
	// TODO: Replace to adaptive colors.
	unStyles = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#DC0100"))
	corsStyles = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFD400"))
)

func Logo(version string) string {
	return lipgloss.JoinVertical(
		lipgloss.Right,
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			unStyles.Render(unLetters),
			corsStyles.Render(corsLetters),
		),
		fmt.Sprintf("version: %s", version),
	)
}
