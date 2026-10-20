package styles

import (
	"image/color"
	"os"

	"charm.land/lipgloss/v2"
	"github.com/evg4b/uncors/internal/events"
)

var lightDark = lipgloss.LightDark(lipgloss.HasDarkBackground(os.Stdin, os.Stdout))

var (
	infoColor     = lightDark(lipgloss.Color("#2563EB"), lipgloss.Color("#3B82F6"))
	successColor  = lightDark(lipgloss.Color("#15803D"), lipgloss.Color("#22C55E"))
	warningColor  = lightDark(lipgloss.Color("#E2A600"), lipgloss.Color("#FFD400"))
	errorColor    = lightDark(lipgloss.Color("#B60000"), lipgloss.Color("#DC0100"))
	contrastColor = lightDark(lipgloss.Color("#FFFFFF"), lipgloss.Color("#000000"))
	dimColor      = lightDark(lipgloss.Color("#6B7280"), lipgloss.Color("#9CA3AF"))

	proxyColor   = lightDark(lipgloss.Color("#4F46E5"), lipgloss.Color("#6366F1"))
	cacheColor   = lightDark(lipgloss.Color("#0E7490"), lipgloss.Color("#06B6D4"))
	mockColor    = lightDark(lipgloss.Color("#C026D3"), lipgloss.Color("#D946EF"))
	scriptColor  = lightDark(lipgloss.Color("#7C3AED"), lipgloss.Color("#8B5CF6"))
	staticColor  = lightDark(lipgloss.Color("#DB2777"), lipgloss.Color("#EC4899"))
	optionsColor = lightDark(lipgloss.Color("#475569"), lipgloss.Color("#94A3B8"))
)

const handlerLabelWidth = 7

var (
	paddedStyle  = lipgloss.NewStyle().Padding(0, 1)
	handlerStyle = paddedStyle.Foreground(contrastColor).Width(handlerLabelWidth)
)

func block(background color.Color) lipgloss.Style {
	return paddedStyle.Background(background).Foreground(contrastColor)
}

var (
	InfoBlockStyle    = block(infoColor)
	WarningBlockStyle = block(warningColor)
	ErrorBlockStyle   = block(errorColor)
	CancelledStyle    = block(dimColor)
	PendingBlockStyle = CancelledStyle.Bold(true)

	URLStyle     = lipgloss.NewStyle().Underline(true)
	DimTextStyle = lipgloss.NewStyle().Foreground(dimColor)

	LogoRed    = lipgloss.NewStyle().Foreground(errorColor)
	LogoYellow = lipgloss.NewStyle().Foreground(warningColor)
)

var statusStyles = map[int]lipgloss.Style{
	1: InfoBlockStyle,
	2: block(successColor),
	3: WarningBlockStyle,
	4: ErrorBlockStyle,
	5: ErrorBlockStyle,
}

func StatusStyle(code int) lipgloss.Style {
	if style, ok := statusStyles[code/100]; ok {
		return style
	}

	return CancelledStyle
}

var handlerStyles = map[events.HandlerKind]lipgloss.Style{
	events.HandlerProxy:   handlerStyle.Background(proxyColor),
	events.HandlerCache:   handlerStyle.Background(cacheColor),
	events.HandlerMock:    handlerStyle.Background(mockColor),
	events.HandlerScript:  handlerStyle.Background(scriptColor),
	events.HandlerStatic:  handlerStyle.Background(staticColor),
	events.HandlerOptions: handlerStyle.Background(optionsColor),
}

func HandlerLabel(kind events.HandlerKind) string {
	style, ok := handlerStyles[kind]
	if !ok {
		return handlerStyle.Render("")
	}

	return style.Render(kind.String())
}
