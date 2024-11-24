package styles

import "github.com/charmbracelet/lipgloss"

var (
	// Logo colors.
	logoYellowColor = lipgloss.AdaptiveColor{
		Light: "#E2A600",
		Dark:  "#FFD400",
	}
	logoRedColor = lipgloss.AdaptiveColor{
		Light: "#B60000",
		Dark:  "#DC0100",
	}

	// Status colors.
	warningColor = lipgloss.AdaptiveColor{
		Light: "#E2A600",
		Dark:  "#FFD400",
	}
	errorColor = lipgloss.AdaptiveColor{
		Light: "#B60000",
		Dark:  "#DC0100",
	}
	contrastColor = lipgloss.AdaptiveColor{
		Light: "#FFFFFF",
		Dark:  "#000000",
	}
	infoColor = lipgloss.AdaptiveColor{
		Light: "#005BA5",
		Dark:  "#0072CE",
	}
	debugColor = lipgloss.AdaptiveColor{
		Light: "#8C8C8C",
		Dark:  "#8C8C8C",
	}

	// Feature colors.
	proxyColor = lipgloss.AdaptiveColor{
		Light: "#545AC9",
		Dark:  "#6A71F7",
	}
	mockColor = lipgloss.AdaptiveColor{
		Light: "#D258DD",
		Dark:  "#EE7FF8",
	}
	staticColor = lipgloss.AdaptiveColor{
		Light: "#588853",
		Dark:  "#A6FC9D",
	}
	cacheColor = lipgloss.AdaptiveColor{
		Light: "#CCC906",
		Dark:  "#FEFC7F",
	}

	// Http status colors.
	httpStatus1xxColor = lipgloss.AdaptiveColor{
		Light: "#005BA5",
		Dark:  "#0072CE",
	}
	httpStatus2xxColor = lipgloss.AdaptiveColor{
		Light: "#01833B",
		Dark:  "#00AF4F",
	}
	httpStatus3xxColor = lipgloss.AdaptiveColor{
		Light: "#E2A600",
		Dark:  "#FFD400",
	}
	httpStatus4xxColor = lipgloss.AdaptiveColor{
		Light: "#B60000",
		Dark:  "#DC0100",
	}
	httpStatus5xxColor = lipgloss.AdaptiveColor{
		Light: "#B60000",
		Dark:  "#DC0100",
	}
)
