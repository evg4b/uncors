package styles

import "github.com/charmbracelet/lipgloss"

const baseBlockWidth = 8

var baseBlock = lipgloss.NewStyle().
	Foreground(contrastColor).
	Padding(0, 1).
	Margin(0).
	Width(baseBlockWidth)

var (
	ProxyStyle  = baseBlock.Background(proxyColor)
	MockStyle   = baseBlock.Background(mockColor)
	StaticStyle = baseBlock.Background(staticColor)
	CacheStyle  = baseBlock.Background(cacheColor)
)
