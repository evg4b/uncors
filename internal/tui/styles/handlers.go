package styles

import "github.com/charmbracelet/lipgloss"

var (
	ProxyStyle = WarningBlock.Copy().
			Background(lipgloss.Color("#6a71f7"))
	MockStyle = WarningBlock.Copy().
			Background(lipgloss.Color("#ee7ff8"))
	StaticStyle = WarningBlock.Copy().
			Background(lipgloss.Color("#ffffff"))
	CacheStyle = WarningBlock.Copy().
			Background(lipgloss.Color("#fefc7f"))
)
