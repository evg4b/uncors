package tui

import (
	"fmt"
	"net/http"
	"strings"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/tui/styles"
	"github.com/evg4b/uncors/pkg/urlt"
)

const (
	RequestPrefixWidth = 11
	methodWidth        = 5
)

func FormatRequest(event events.RequestCompleted) string {
	var builder strings.Builder

	builder.WriteString(styles.HandlerLabel(event.Handler))
	builder.WriteString(printResponse(event.Data))

	if event.Rewrite != nil {
		builder.WriteByte(' ')
		builder.WriteString(styles.DimTextStyle.Render("→ " + event.Rewrite.To))
	}

	if event.Err != nil && !event.Data.Cancelled {
		builder.WriteString("\n  ")
		builder.WriteString(styles.DimTextStyle.Render("↳ " + event.Err.Error()))
	}

	return builder.String()
}

func printResponse(data *contracts.RequestData) string {
	var (
		prefix                 string
		prefixStyle, textStyle lipgloss.Style
	)

	if data.Cancelled {
		prefix = fmt.Sprintf("CXL %s", data.Method)
		prefixStyle = styles.HTTPStatusCancelledBlockStyle
		textStyle = styles.HTTPStatusCancelledTextStyle
	} else {
		prefix = fmt.Sprintf("%d %s", data.Code, MethodLabel(data.Method))
		prefixStyle, textStyle = getStyles(data.Code)
	}

	prefixStyle = prefixStyle.Width(RequestPrefixWidth)

	return fmt.Sprintf("%s %s", prefixStyle.Render(prefix), textStyle.Render(urlt.URL_String(data.URL)))
}

var methodLabels = map[string]string{
	http.MethodDelete:  "DEL",
	http.MethodOptions: "OPTS",
	http.MethodConnect: "CONN",
}

func MethodLabel(method string) string {
	if label, ok := methodLabels[method]; ok {
		return label
	}

	return ansi.Truncate(method, methodWidth, "…")
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

	return styles.HTTPStatusOtherBlockStyle, styles.HTTPStatusOtherTextStyle
}
