package styles

import (
	"os"

	lipgloss "charm.land/lipgloss/v2"
)

var lightDark = lipgloss.LightDark(lipgloss.HasDarkBackground(os.Stdin, os.Stdout))

var (
	// Logo colors.

	logoYellowColor = lightDark(lipgloss.Color("#E2A600"), lipgloss.Color("#FFD400"))
	logoRedColor    = lightDark(lipgloss.Color("#B60000"), lipgloss.Color("#DC0100"))

	// Status colors.

	WarningColor  = lightDark(lipgloss.Color("#E2A600"), lipgloss.Color("#FFD400"))
	ErrorColor    = lightDark(lipgloss.Color("#B60000"), lipgloss.Color("#DC0100"))
	ContrastColor = lightDark(lipgloss.Color("#FFFFFF"), lipgloss.Color("#000000"))
	InfoColor     = lightDark(lipgloss.Color("#005BA5"), lipgloss.Color("#0072CE"))
	DebugColor    = lightDark(lipgloss.Color("#8C8C8C"), lipgloss.Color("#8C8C8C"))

	// Feature colors.

	proxyColor   = lightDark(lipgloss.Color("#545AC9"), lipgloss.Color("#6A71F7"))
	mockColor    = lightDark(lipgloss.Color("#D258DD"), lipgloss.Color("#EE7FF8"))
	staticColor  = lightDark(lipgloss.Color("#588853"), lipgloss.Color("#A6FC9D"))
	cacheColor   = lightDark(lipgloss.Color("#CCC906"), lipgloss.Color("#FEFC7F"))
	rewriteColor = lightDark(lipgloss.Color("#FF7F00"), lipgloss.Color("#FF7F00"))
	optionsColor = lightDark(lipgloss.Color("#005BA5"), lipgloss.Color("#0072CE"))

	// Http status colors.

	httpStatus1xxColor = lightDark(lipgloss.Color("#005BA5"), lipgloss.Color("#0072CE"))
	httpStatus2xxColor = lightDark(lipgloss.Color("#01833B"), lipgloss.Color("#00AF4F"))
	httpStatus3xxColor = lightDark(lipgloss.Color("#E2A600"), lipgloss.Color("#FFD400"))
	httpStatus4xxColor = lightDark(lipgloss.Color("#B60000"), lipgloss.Color("#DC0100"))
	httpStatus5xxColor = lightDark(lipgloss.Color("#B60000"), lipgloss.Color("#DC0100"))
)
