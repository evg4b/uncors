package styles

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

const handlerLabelWidth = 7

var paddedStyle = lipgloss.NewStyle().Padding(0, 1)

func blockStyle(palette Palette, background color.Color) lipgloss.Style {
	return paddedStyle.Background(background).Foreground(palette.Contrast)
}

func handlerStyle(palette Palette) lipgloss.Style {
	return paddedStyle.Foreground(palette.Contrast).Width(handlerLabelWidth)
}

func textStyle(foreground color.Color) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(foreground)
}
