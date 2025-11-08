package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/evg4b/uncors/internal/tui/styles"
)

func PrintWarningBox(out io.Writer, messages ...string) {
	printMessageBox(
		out,
		strings.Join(messages, "\n"),
		warningLabel,
		styles.WarningBlockStyle,
	)
}

func PrintInfoBox(out io.Writer, messages ...string) {
	printMessageBox(
		out,
		strings.Join(messages, "\n"),
		infoLabel,
		styles.InfoBlockStyle,
	)
}

func PrintErrorBox(out io.Writer, messages ...string) {
	printMessageBox(
		out,
		strings.Join(messages, "\n"),
		errorLabel,
		styles.ErrorBlockStyle,
	)
}

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
