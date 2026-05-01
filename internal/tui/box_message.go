package tui

import (
	"fmt"
	"io"
	"strings"

	lipgloss "charm.land/lipgloss/v2"
)

func printMessageBox(out io.Writer, message, prefix string, blockStyles lipgloss.Style) {
	height := lipgloss.Height(message)
	space := strings.Repeat("\n", height-1)

	blockStyles = blockStyles.Margin(0, 1, 0, 0)
	block := lipgloss.JoinHorizontal(
		lipgloss.Top,
		blockStyles.Render(prefix, space),
		message,
	)

	_, err := fmt.Fprintln(out, block)
	if err != nil {
		panic(err)
	}
}
