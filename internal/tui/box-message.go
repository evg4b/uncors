package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/evg4b/uncors/internal/tui/styles"
)

func PrintWarningBox(out io.Writer, message string) {
	printMessageBox(
		out,
		message,
		styles.WarningLabel,
		styles.WarningBlock,
	)
}

func PrintInfoBox(out io.Writer, message string) {
	printMessageBox(
		out,
		message,
		styles.InfoLabel,
		styles.InfoBlock,
	)
}

func printMessageBox(out io.Writer, message, prefix string, block lipgloss.Style) {
	height := lipgloss.Height(message)
	space := strings.Repeat("\n", height-1)

	block = block.Copy().Margin(0, 1, 0, 0)

	_, err := fmt.Fprintln(out, lipgloss.JoinHorizontal( //nolint:forbidigo
		lipgloss.Top,
		block.Render(prefix, space),
		message,
	))
	if err != nil {
		panic(err)
	}
}
