package request_tracker

import (
	"strconv"

	"github.com/charmbracelet/lipgloss"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/tui/styles"
)

func RenderDoneRequest(request DoneRequestDefinition) string {
	style := selectStyles(request.Status)

	return render(request.RequestDefinition, formatCode(request), style)
}

func formatCode(request DoneRequestDefinition) string {
	if request.Status == 0 {
		return "✖✖✖"
	}

	return strconv.Itoa(request.Status)
}

func RenderRequest(request RequestDefinition, spinner string) string {
	return render(request, spinner, styles.PendingStyle)
}

func render(request RequestDefinition, status string, style styles.StatusStyle) string {
	method := lipgloss.PlaceHorizontal(4, lipgloss.Left, request.Method)

	return lipgloss.JoinHorizontal(
		lipgloss.Left,
		request.Type,
		style.BlockStyle.Render(method+" "+status),
		style.MainTextStyle.Render(request.Host),
		style.MainTextStyle.Render(request.Path),
	)
}

func selectStyles(status int) styles.StatusStyle {
	switch {
	case helpers.Is1xxCode(status):
		return styles.InformationalStyle
	case helpers.Is2xxCode(status):
		return styles.SuccessStyle
	case helpers.Is3xxCode(status):
		return styles.RedirectionStyle
	case helpers.Is4xxCode(status):
		return styles.ClientErrorStyle
	case helpers.Is5xxCode(status):
		return styles.ServerErrorStyle
	default:
		return styles.CanceledStyle
	}
}
