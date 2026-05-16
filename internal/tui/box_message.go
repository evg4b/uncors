package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

func (output *CliOutput) printMessageBox(message, prefix string, blockStyles lipgloss.Style) {
	height := lipgloss.Height(message)
	space := strings.Repeat("\n", height-1)

	blockStyles = blockStyles.Margin(0, 1, 0, 0)
	block := lipgloss.JoinHorizontal(
		lipgloss.Top,
		blockStyles.Render(prefix, space),
		message,
	)

	_, err := fmt.Fprintln(output.output, block)
	if err != nil {
		panic(err)
	}
}
