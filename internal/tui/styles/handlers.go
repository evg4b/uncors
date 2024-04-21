package styles

import "github.com/charmbracelet/lipgloss"

const baseBlockWidth = 8

var baseBlock = lipgloss.NewStyle().
	Foreground(black).
	Padding(0, 1).
	Margin(0).
	Width(baseBlockWidth).
	ColorWhitespace(true)

var (
	ProxyStyle = baseBlock.Copy().
			Background(lipgloss.Color("#6a71f7"))

	MockStyle = baseBlock.Copy().
			Background(lipgloss.Color("#ee7ff8"))

	StaticStyle = baseBlock.Copy().
			Background(lipgloss.Color("#ffffff"))

	CacheStyle = baseBlock.Copy().
			Background(lipgloss.Color("#fefc7f"))
)
