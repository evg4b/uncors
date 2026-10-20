package styles

import (
	"image/color"
	"os"

	"charm.land/lipgloss/v2"
	"github.com/evg4b/uncors/internal/events"
)

const handlerLabelWidth = 7

var paddedStyle = lipgloss.NewStyle().Padding(0, 1)

type Theme struct {
	InfoBlock    lipgloss.Style
	WarningBlock lipgloss.Style
	ErrorBlock   lipgloss.Style
	Cancelled    lipgloss.Style
	PendingBlock lipgloss.Style
	URL          lipgloss.Style
	DimText      lipgloss.Style
	MutedText    lipgloss.Style
	LogoRed      lipgloss.Style
	LogoYellow   lipgloss.Style

	handler  lipgloss.Style
	handlers map[events.HandlerKind]lipgloss.Style
	statuses map[int]lipgloss.Style
}

func NewTheme(dark bool) Theme {
	lightDark := lipgloss.LightDark(dark)

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

	block := func(background color.Color) lipgloss.Style {
		return paddedStyle.Background(background).Foreground(contrastColor)
	}

	handler := paddedStyle.Foreground(contrastColor).Width(handlerLabelWidth)

	return Theme{
		InfoBlock:    block(infoColor),
		WarningBlock: block(warningColor),
		ErrorBlock:   block(errorColor),
		Cancelled:    block(dimColor),
		PendingBlock: paddedStyle.Background(dimColor).Foreground(whiteColor).Bold(true),
		URL:          lipgloss.NewStyle().Underline(true),
		DimText:      lipgloss.NewStyle().Foreground(dimColor),
		MutedText:    lipgloss.NewStyle().Foreground(mutedColor),
		LogoRed:      lipgloss.NewStyle().Foreground(errorColor),
		LogoYellow:   lipgloss.NewStyle().Foreground(warningColor),

		handler: handler,
		handlers: map[events.HandlerKind]lipgloss.Style{
			events.HandlerProxy:   handler.Background(proxyColor),
			events.HandlerCache:   handler.Background(cacheColor),
			events.HandlerMock:    handler.Background(mockColor),
			events.HandlerScript:  handler.Background(scriptColor),
			events.HandlerStatic:  handler.Background(staticColor),
			events.HandlerOptions: handler.Background(optionsColor),
		},
		statuses: map[int]lipgloss.Style{
			1: block(infoColor),
			2: block(successColor),
			3: block(warningColor),
			4: block(errorColor),
			5: block(errorColor),
		},
	}
}

func (t Theme) StatusStyle(code int) lipgloss.Style {
	if style, ok := t.statuses[code/100]; ok {
		return style
	}

	return t.Cancelled
}

func (t Theme) HandlerLabel(kind events.HandlerKind) string {
	style, ok := t.handlers[kind]
	if !ok {
		return t.handler.Render("")
	}

	return style.Render(kind.String())
}

var Default = NewTheme(lipgloss.HasDarkBackground(os.Stdin, os.Stdout))

var (
	InfoBlockStyle    = Default.InfoBlock
	WarningBlockStyle = Default.WarningBlock
	ErrorBlockStyle   = Default.ErrorBlock
	CancelledStyle    = Default.Cancelled
	PendingBlockStyle = Default.PendingBlock
	URLStyle          = Default.URL
	DimTextStyle      = Default.DimText
	MutedTextStyle    = Default.MutedText
	LogoRed           = Default.LogoRed
	LogoYellow        = Default.LogoYellow
)

func StatusStyle(code int) lipgloss.Style {
	return Default.StatusStyle(code)
}

func HandlerLabel(kind events.HandlerKind) string {
	return Default.HandlerLabel(kind)
}
