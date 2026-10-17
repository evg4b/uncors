package styles

import (
	"os"

	"charm.land/lipgloss/v2"
)

var lightDark = lipgloss.LightDark(lipgloss.HasDarkBackground(os.Stdin, os.Stdout))

var (
	WarningColor  = lightDark(lipgloss.Color("#E2A600"), lipgloss.Color("#FFD400"))
	ErrorColor    = lightDark(lipgloss.Color("#B60000"), lipgloss.Color("#DC0100"))
	ContrastColor = lightDark(lipgloss.Color("#FFFFFF"), lipgloss.Color("#000000"))
	InfoColor     = lightDark(lipgloss.Color("#005BA5"), lipgloss.Color("#0072CE"))
	DebugColor    = lipgloss.Color("#8C8C8C")

	logoYellowColor = WarningColor
	logoRedColor    = ErrorColor

	proxyColor   = lightDark(lipgloss.Color("#545AC9"), lipgloss.Color("#6A71F7"))
	mockColor    = lightDark(lipgloss.Color("#D258DD"), lipgloss.Color("#EE7FF8"))
	staticColor  = lightDark(lipgloss.Color("#588853"), lipgloss.Color("#A6FC9D"))
	cacheColor   = lightDark(lipgloss.Color("#CCC906"), lipgloss.Color("#FEFC7F"))
	scriptColor  = lipgloss.Color("#FF7F00")
	optionsColor = InfoColor

	httpStatus1xxColor       = InfoColor
	httpStatus2xxColor       = lightDark(lipgloss.Color("#01833B"), lipgloss.Color("#00AF4F"))
	httpStatus3xxColor       = WarningColor
	httpStatus4xxColor       = ErrorColor
	httpStatus5xxColor       = ErrorColor
	httpStatusCancelledColor = DebugColor
	httpStatusOtherColor     = DebugColor
)
