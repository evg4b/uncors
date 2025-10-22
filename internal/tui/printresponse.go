package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/tui/styles"
)

const prefixWidth = 13

func printResponse(request *contracts.Request, statusCode int) string {
	prefix := fmt.Sprintf("%d %s", statusCode, request.Method)
	prefixStyle, textStyle := getStyles(statusCode)
	prefixStyle = prefixStyle.Width(prefixWidth)

	return fmt.Sprintf("%s %s", prefixStyle.Render(prefix), textStyle.Render(request.URL.String()))
}

func getStyles(statusCode int) (lipgloss.Style, lipgloss.Style) {
	if helpers.Is1xxCode(statusCode) {
		return styles.HTTPStatus1xxBlockStyle, styles.HTTPStatus1xxTextStyle
	}

	if helpers.Is2xxCode(statusCode) {
		return styles.HTTPStatus2xxBlockStyle, styles.HTTPStatus2xxTextStyle
	}

	if helpers.Is3xxCode(statusCode) {
		return styles.HTTPStatus3xxBlockStyle, styles.HTTPStatus3xxTextStyle
	}

	if helpers.Is4xxCode(statusCode) {
		return styles.HTTPStatus4xxBlockStyle, styles.HTTPStatus4xxTextStyle
	}

	if helpers.Is5xxCode(statusCode) {
		return styles.HTTPStatus5xxBlockStyle, styles.HTTPStatus5xxTextStyle
	}

	panic(fmt.Sprintf("status code %d is not supported", statusCode))
}
