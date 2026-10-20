package styles

import (
	"image/color"
	"os"

	"charm.land/lipgloss/v2"
	"github.com/evg4b/uncors/internal/events"
)

var lightDark = lipgloss.LightDark(lipgloss.HasDarkBackground(os.Stdin, os.Stdout))

var (
	infoColor     = lightDark(lipgloss.Color("#005BA5"), lipgloss.Color("#0072CE"))
	successColor  = lightDark(lipgloss.Color("#01833B"), lipgloss.Color("#00AF4F"))
	warningColor  = lightDark(lipgloss.Color("#E2A600"), lipgloss.Color("#FFD400"))
	errorColor    = lightDark(lipgloss.Color("#B60000"), lipgloss.Color("#DC0100"))
	contrastColor = lightDark(lipgloss.Color("#FFFFFF"), lipgloss.Color("#000000"))
	dimColor      = lipgloss.Color("#8C8C8C")
	mutedColor    = lipgloss.Color("#777777")
	whiteColor    = lipgloss.Color("#FFFFFF")

	proxyColor   = lightDark(lipgloss.Color("#545AC9"), lipgloss.Color("#6A71F7"))
	cacheColor   = lightDark(lipgloss.Color("#CCC906"), lipgloss.Color("#FEFC7F"))
	mockColor    = lightDark(lipgloss.Color("#D258DD"), lipgloss.Color("#EE7FF8"))
	scriptColor  = lipgloss.Color("#FF7F00")
	staticColor  = lightDark(lipgloss.Color("#588853"), lipgloss.Color("#A6FC9D"))
	optionsColor = lightDark(lipgloss.Color("#384957"), lipgloss.Color("#C5D1DB"))
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
	PendingBlockStyle = paddedStyle.Background(dimColor).Foreground(whiteColor).Bold(true)

	URLStyle       = lipgloss.NewStyle().Underline(true)
	DimTextStyle   = lipgloss.NewStyle().Foreground(dimColor)
	MutedTextStyle = lipgloss.NewStyle().Foreground(mutedColor)

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
