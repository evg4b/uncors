package render

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/tui/styles"
	"github.com/evg4b/uncors/pkg/urlt"
)

const (
	requestPrefixWidth = 11
	methodWidth        = 5
)

func FormatRequest(theme styles.Theme, event events.RequestCompleted) string {
	var builder strings.Builder

	builder.WriteString(theme.HandlerLabel(event.Handler))
	builder.WriteString(printResponse(theme, event.Data))
	writeRewrite(theme, &builder, event.Rewrite)

	if event.Err != nil && !event.Data.Cancelled {
		builder.WriteString("\n  ")
		builder.WriteString(theme.DimText.Render("↳ " + event.Err.Error()))
	}

	return builder.String()
}

func FormatPending(theme styles.Theme, event events.RequestStarted, frame string) string {
	var builder strings.Builder

	urlStr := ""
	if event.URL != nil {
		urlStr = urlt.URL_String(event.URL)
	}

	builder.WriteString(theme.HandlerLabel(event.Handler))
	builder.WriteString(theme.PendingBlock.Width(requestPrefixWidth).Render(frame + " " + methodLabel(event.Method)))
	builder.WriteByte(' ')
	builder.WriteString(theme.DimText.Render(urlStr))
	writeRewrite(theme, &builder, event.Rewrite)

	return builder.String()
}

func writeRewrite(theme styles.Theme, builder *strings.Builder, rewrite *events.Rewrite) {
	if rewrite != nil {
		builder.WriteByte(' ')
		builder.WriteString(theme.DimText.Render("→ " + rewrite.To))
	}
}

func printResponse(theme styles.Theme, data *contracts.RequestData) string {
	prefix := fmt.Sprintf("%d %s", data.Code, methodLabel(data.Method))
	prefixStyle := theme.StatusStyle(data.Code)

	if data.Cancelled {
		prefix = "CXL " + methodLabel(data.Method)
		prefixStyle = theme.Cancelled
	}

	return prefixStyle.Width(requestPrefixWidth).Render(prefix) + " " + theme.URL.Render(urlt.URL_String(data.URL))
}

var methodLabels = map[string]string{
	http.MethodDelete:  "DEL",
	http.MethodOptions: "OPTS",
	http.MethodConnect: "CONN",
}

func methodLabel(method string) string {
	if label, ok := methodLabels[method]; ok {
		return label
	}

	return ansi.Truncate(method, methodWidth, "…")
}
