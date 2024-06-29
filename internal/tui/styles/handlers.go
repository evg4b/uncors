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
	ProxyStyle  = baseBlock.Background(lipgloss.Color("#6a71f7"))
	MockStyle   = baseBlock.Background(lipgloss.Color("#ee7ff8"))
	StaticStyle = baseBlock.Background(lipgloss.Color("#ffffff"))
	CacheStyle  = baseBlock.Background(lipgloss.Color("#fefc7f"))
)
