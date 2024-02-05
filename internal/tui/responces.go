package tui

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/tui/styles"
)

func RenderDoneRequest(request DoneRequestDefinition) string {
	block, text := selectStyles(request.Status)

	return render(request.RequestDefinition, strconv.Itoa(request.Status), block, text)
}

func RenderRequest(request RequestDefinition, spinner string) string {
	return render(request, spinner, styles.DisabledBlock, styles.DisabledText)
}

func render(request RequestDefinition, status string, block, text lipgloss.Style) string {
	return lipgloss.JoinHorizontal(
		lipgloss.Left,
		block.Render(fmt.Sprintf("%s %s", request.Method, status)),
		text.Render(request.URL),
	)
}

func selectStyles(status int) (lipgloss.Style, lipgloss.Style) {
	switch {
	case helpers.Is1xxCode(status):
		return styles.WarningBlock, styles.WarningText
	case helpers.Is2xxCode(status):
		return styles.SuccessBlock, styles.SuccessText
	case helpers.Is3xxCode(status):
		return styles.WarningBlock, styles.WarningText
	case helpers.Is4xxCode(status), helpers.Is5xxCode(status):
		return styles.ErrorBlock, styles.ErrorText
	}

	log.Warnf("Unknown status code %d", status)

	return styles.DisabledBlock, styles.DisabledText
}
