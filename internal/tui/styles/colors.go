package styles

import (
	lipgloss "charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
)

var (
	// Logo colors.
	logoYellowColor = compat.AdaptiveColor{
		Light: lipgloss.Color("#E2A600"),
		Dark:  lipgloss.Color("#FFD400"),
	}
	logoRedColor = compat.AdaptiveColor{
		Light: lipgloss.Color("#B60000"),
		Dark:  lipgloss.Color("#DC0100"),
	}

	// Status colors.
	warningColor = compat.AdaptiveColor{
		Light: lipgloss.Color("#E2A600"),
		Dark:  lipgloss.Color("#FFD400"),
	}
	errorColor = compat.AdaptiveColor{
		Light: lipgloss.Color("#B60000"),
		Dark:  lipgloss.Color("#DC0100"),
	}
	contrastColor = compat.AdaptiveColor{
		Light: lipgloss.Color("#FFFFFF"),
		Dark:  lipgloss.Color("#000000"),
	}
	infoColor = compat.AdaptiveColor{
		Light: lipgloss.Color("#005BA5"),
		Dark:  lipgloss.Color("#0072CE"),
	}
	debugColor = compat.AdaptiveColor{
		Light: lipgloss.Color("#8C8C8C"),
		Dark:  lipgloss.Color("#8C8C8C"),
	}

	// Feature colors.
	proxyColor = compat.AdaptiveColor{
		Light: lipgloss.Color("#545AC9"),
		Dark:  lipgloss.Color("#6A71F7"),
	}
	mockColor = compat.AdaptiveColor{
		Light: lipgloss.Color("#D258DD"),
		Dark:  lipgloss.Color("#EE7FF8"),
	}
	staticColor = compat.AdaptiveColor{
		Light: lipgloss.Color("#588853"),
		Dark:  lipgloss.Color("#A6FC9D"),
	}
	cacheColor = compat.AdaptiveColor{
		Light: lipgloss.Color("#CCC906"),
		Dark:  lipgloss.Color("#FEFC7F"),
	}
	rewriteColor = compat.AdaptiveColor{
		Light: lipgloss.Color("#FF7F00"),
		Dark:  lipgloss.Color("#FF7F00"),
	}
	optionsColor = compat.AdaptiveColor{
		Light: lipgloss.Color("#005BA5"),
		Dark:  lipgloss.Color("#0072CE"),
	}

	// Http status colors.
	httpStatus1xxColor = compat.AdaptiveColor{
		Light: lipgloss.Color("#005BA5"),
		Dark:  lipgloss.Color("#0072CE"),
	}
	httpStatus2xxColor = compat.AdaptiveColor{
		Light: lipgloss.Color("#01833B"),
		Dark:  lipgloss.Color("#00AF4F"),
	}
	httpStatus3xxColor = compat.AdaptiveColor{
		Light: lipgloss.Color("#E2A600"),
		Dark:  lipgloss.Color("#FFD400"),
	}
	httpStatus4xxColor = compat.AdaptiveColor{
		Light: lipgloss.Color("#B60000"),
		Dark:  lipgloss.Color("#DC0100"),
	}
	httpStatus5xxColor = compat.AdaptiveColor{
		Light: lipgloss.Color("#B60000"),
		Dark:  lipgloss.Color("#DC0100"),
	}
)
