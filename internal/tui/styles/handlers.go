package styles

import "github.com/charmbracelet/lipgloss"

const blockWidth = 8

var (
	ProxyStyle = WarningBlock.Copy().
			Background(lipgloss.Color("#6a71f7")).
			Width(blockWidth)
	MockStyle = WarningBlock.Copy().
			Background(lipgloss.Color("#ee7ff8")).
			Width(blockWidth)
	StaticStyle = WarningBlock.Copy().
			Background(lipgloss.Color("#ffffff")).
			Width(blockWidth)
	CacheStyle = WarningBlock.Copy().
			Background(lipgloss.Color("#fefc7f")).
			Width(blockWidth)
)
