package styles

import "github.com/charmbracelet/lipgloss"

var (
	DebugText  = lipgloss.NewStyle()
	DebugBlock = lipgloss.NewStyle().
			Background(debugColor).
			Foreground(contrastColor).
			Padding(0, 1)
	WarningText  = lipgloss.NewStyle()
	WarningBlock = lipgloss.NewStyle().
			Background(warningColor).
			Foreground(contrastColor).
			Padding(0, 1)
	InfoText  = lipgloss.NewStyle()
	InfoBlock = lipgloss.NewStyle().
			Background(infoColor).
			Foreground(contrastColor).
			Padding(0, 1)
	ErrorText  = lipgloss.NewStyle()
	ErrorBlock = lipgloss.NewStyle().
			Background(errorColor).
			Foreground(contrastColor).
			Padding(0, 1)
)
