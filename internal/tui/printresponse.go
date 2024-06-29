package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/tui/styles"
)

const prefixWidth = 12

func printResponse(request *contracts.Request, statusCode int) string {
	prefix := helpers.Sprintf("%d %s", statusCode, request.Method)
	prefixStyle, textStyle := getPrefixPrinter(statusCode)
	prefixStyle = prefixStyle.Width(prefixWidth)

	return fmt.Sprintf("%s %s", prefixStyle.Render(prefix), textStyle.Render(request.URL.String()))
}

func getPrefixPrinter(statusCode int) (lipgloss.Style, lipgloss.Style) {
	if helpers.Is1xxCode(statusCode) {
		return styles.InfoBlock, styles.InfoText
	}

	if helpers.Is2xxCode(statusCode) {
		return styles.SuccessBlock, styles.SuccessText
	}

	if helpers.Is3xxCode(statusCode) {
		return styles.WarningBlock, styles.WarningText
	}

	if helpers.Is4xxCode(statusCode) || helpers.Is5xxCode(statusCode) {
		return styles.ErrorBlock, styles.ErrorText
	}

	panic(helpers.Sprintf("status code %d is not supported", statusCode))
}
