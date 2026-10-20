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

func FormatRequest(event events.RequestCompleted) string {
	var builder strings.Builder

	builder.WriteString(styles.HandlerLabel(event.Handler))
	builder.WriteString(printResponse(event.Data))
	writeRewrite(&builder, event.Rewrite)

	if event.Err != nil && !event.Data.Cancelled {
		builder.WriteString("\n  ")
		builder.WriteString(styles.DimTextStyle.Render("↳ " + event.Err.Error()))
	}

	return builder.String()
}

func FormatPending(event events.RequestStarted, frame string) string {
	var builder strings.Builder

	urlStr := ""
	if event.URL != nil {
		urlStr = urlt.URL_String(event.URL)
	}

	builder.WriteString(styles.HandlerLabel(event.Handler))
	builder.WriteString(styles.PendingBlockStyle.Width(requestPrefixWidth).Render(frame + " " + methodLabel(event.Method)))
	builder.WriteByte(' ')
	builder.WriteString(styles.DimTextStyle.Render(urlStr))
	writeRewrite(&builder, event.Rewrite)

	return builder.String()
}

func writeRewrite(builder *strings.Builder, rewrite *events.Rewrite) {
	if rewrite != nil {
		builder.WriteByte(' ')
		builder.WriteString(styles.DimTextStyle.Render("→ " + rewrite.To))
	}
}

func printResponse(data *contracts.RequestData) string {
	prefix := fmt.Sprintf("%d %s", data.Code, methodLabel(data.Method))
	prefixStyle := styles.StatusStyle(data.Code)

	if data.Cancelled {
		prefix = "CXL " + methodLabel(data.Method)
		prefixStyle = styles.CancelledStyle
	}

	return prefixStyle.Width(requestPrefixWidth).Render(prefix) + " " + styles.URLStyle.Render(urlt.URL_String(data.URL))
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
