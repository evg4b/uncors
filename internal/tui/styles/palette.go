package styles

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

type Palette struct {
	Info     color.Color
	Success  color.Color
	Warning  color.Color
	Error    color.Color
	Contrast color.Color
	Dim      color.Color
	Muted    color.Color
	White    color.Color

	Proxy   color.Color
	Cache   color.Color
	Mock    color.Color
	Script  color.Color
	Static  color.Color
	Options color.Color
}

var Light = Palette{
	Info:     lipgloss.Color("#005BA5"),
	Success:  lipgloss.Color("#01833B"),
	Warning:  lipgloss.Color("#E2A600"),
	Error:    lipgloss.Color("#B60000"),
	Contrast: lipgloss.Color("#FFFFFF"),
	Dim:      lipgloss.Color("#8C8C8C"),
	Muted:    lipgloss.Color("#777777"),
	White:    lipgloss.Color("#FFFFFF"),

	Proxy:   lipgloss.Color("#545AC9"),
	Cache:   lipgloss.Color("#CCC906"),
	Mock:    lipgloss.Color("#D258DD"),
	Script:  lipgloss.Color("#FF7F00"),
	Static:  lipgloss.Color("#588853"),
	Options: lipgloss.Color("#384957"),
}

var Dark = Palette{
	Info:     lipgloss.Color("#0072CE"),
	Success:  lipgloss.Color("#00AF4F"),
	Warning:  lipgloss.Color("#FFD400"),
	Error:    lipgloss.Color("#DC0100"),
	Contrast: lipgloss.Color("#000000"),
	Dim:      lipgloss.Color("#8C8C8C"),
	Muted:    lipgloss.Color("#777777"),
	White:    lipgloss.Color("#FFFFFF"),

	Proxy:   lipgloss.Color("#6A71F7"),
	Cache:   lipgloss.Color("#FEFC7F"),
	Mock:    lipgloss.Color("#EE7FF8"),
	Script:  lipgloss.Color("#FF7F00"),
	Static:  lipgloss.Color("#A6FC9D"),
	Options: lipgloss.Color("#C5D1DB"),
}
