package log

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
)

type colorScheme struct {
	prefix lipgloss.Style
	url    lipgloss.Style
	params lipgloss.Style
}

const (
	// TODO: Replace to adaptive colors.
	yellow = lipgloss.Color("#c6c400")
	red    = lipgloss.Color("#ff6e67")
	black  = lipgloss.Color("0")
	green  = lipgloss.Color("#00c202")
	cyan   = lipgloss.Color("#03c5c7")
)

var prefixStyle = lipgloss.NewStyle().
	Foreground(black).
	PaddingLeft(1).
	PaddingRight(1)

var errorScheme = colorScheme{
	prefix: prefixStyle.Copy().Background(red),
	url:    lipgloss.NewStyle().Foreground(red),
	params: lipgloss.NewStyle().Foreground(lipgloss.Color("#7a3330")),
}

var warningScheme = colorScheme{
	prefix: prefixStyle.Copy().Background(yellow),
	url:    lipgloss.NewStyle().Foreground(yellow),
	params: lipgloss.NewStyle().Foreground(lipgloss.Color("#777600")),
}

var successScheme = colorScheme{
	prefix: prefixStyle.Copy().Background(green),
	url:    lipgloss.NewStyle().Foreground(green),
	params: lipgloss.NewStyle().Foreground(lipgloss.Color("#004e01")),
}

var infoScheme = colorScheme{
	prefix: prefixStyle.Copy().
		Background(cyan),
	url: lipgloss.NewStyle().
		Foreground(cyan),
	params: lipgloss.NewStyle().
		Foreground(lipgloss.Color("#027677")),
}

func printResponse(request *contracts.Request, statusCode int) string {
	prefix := helpers.Sprintf("%d %s", statusCode, request.Method)
	scheme := getScheme(statusCode)

	return fmt.Sprintf("%s %s", scheme.prefix.Render(prefix), formatURL(request, scheme))
}

func getScheme(statusCode int) colorScheme {
	switch {
	case helpers.Is4xxCode(statusCode) || helpers.Is5xxCode(statusCode):
		return errorScheme
	case helpers.Is3xxCode(statusCode):
		return warningScheme
	case helpers.Is2xxCode(statusCode):
		return successScheme
	case helpers.Is1xxCode(statusCode):
		return infoScheme
	default:
		panic(helpers.Sprintf("status code %d is not supported", statusCode))
	}
}

func formatURL(request *contracts.Request, scheme colorScheme) string {
	url := request.URL.String()
	if request.URL.RawQuery == "" {
		return scheme.url.Render(url)
	}

	parts := strings.Split(url, "?")

	return fmt.Sprintf("%s%s", scheme.url.Render(parts[0]), scheme.params.Render("?"+parts[1]))
}
