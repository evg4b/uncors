package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/evg4b/uncors/internal/tui/styles"
)

func PrintWarningBox(message string) {
	printMessageBox(message, styles.WarningLabel, styles.WarningBlock)
}

func PrintInfoBox(message string) {
	printMessageBox(message, styles.InfoLabel, styles.InfoBlock)
}

func printMessageBox(message, prefix string, block lipgloss.Style) {
	height := lipgloss.Height(message)
	space := strings.Repeat("\n", height-1)

	block = block.Copy().Margin(0, 1, 0, 0)

	println(lipgloss.JoinHorizontal( //nolint:forbidigo
		lipgloss.Top,
		block.Render(prefix, space),
		message,
	))
}
