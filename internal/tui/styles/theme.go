package styles

import (
	"image/color"
	"os"

	"charm.land/lipgloss/v2"
	"github.com/evg4b/uncors/internal/events"
)

const statusClassSize = 100

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

	handlerLabels [events.HandlerOptions + 1]string
	statuses      [6]lipgloss.Style
}

func NewTheme(palette Palette) *Theme {
	block := func(background color.Color) lipgloss.Style {
		return blockStyle(palette, background)
	}

	theme := &Theme{
		InfoBlock:    block(palette.Info),
		WarningBlock: block(palette.Warning),
		ErrorBlock:   block(palette.Error),
		Cancelled:    block(palette.Dim),
		PendingBlock: paddedStyle.Background(palette.Dim).Foreground(palette.White).Bold(true),
		URL:          lipgloss.NewStyle().Underline(true),
		DimText:      textStyle(palette.Dim),
		MutedText:    textStyle(palette.Muted),
		LogoRed:      textStyle(palette.Error),
		LogoYellow:   textStyle(palette.Warning),
	}

	theme.statuses = [...]lipgloss.Style{
		theme.Cancelled,
		block(palette.Info),
		block(palette.Success),
		block(palette.Warning),
		block(palette.Error),
		block(palette.Error),
	}

	handler := handlerStyle(palette)
	handlerColors := [...]color.Color{
		events.HandlerProxy:   palette.Proxy,
		events.HandlerCache:   palette.Cache,
		events.HandlerMock:    palette.Mock,
		events.HandlerScript:  palette.Script,
		events.HandlerStatic:  palette.Static,
		events.HandlerOptions: palette.Options,
	}

	for kind, background := range handlerColors {
		if background == nil {
			theme.handlerLabels[kind] = handler.Render("")

			continue
		}

		theme.handlerLabels[kind] = handler.Background(background).Render(events.HandlerKind(kind).String())
	}

	return theme
}

func (t *Theme) StatusStyle(code int) lipgloss.Style {
	if index := code / statusClassSize; index >= 0 && index < len(t.statuses) {
		return t.statuses[index]
	}

	return t.Cancelled
}

func (t *Theme) HandlerLabel(kind events.HandlerKind) string {
	if int(kind) < len(t.handlerLabels) {
		return t.handlerLabels[kind]
	}

	return t.handlerLabels[events.HandlerNone]
}

func PaletteFor(dark bool) Palette {
	if dark {
		return Dark
	}

	return Light
}

var Default = NewTheme(PaletteFor(lipgloss.HasDarkBackground(os.Stdin, os.Stdout)))
