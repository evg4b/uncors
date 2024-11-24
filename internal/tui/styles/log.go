package styles

import "github.com/charmbracelet/lipgloss"

var (
	LogoYellow = lipgloss.NewStyle().
			Foreground(logoYellowColor)

	LogoRed = lipgloss.NewStyle().
		Foreground(logoRedColor)

	WarningBlock = lipgloss.NewStyle().
			Background(warningColor).
			Foreground(textColor).
			Padding(0, 1)

	WarningText = lipgloss.NewStyle().
			Foreground(warningColor)

	InfoBlock = lipgloss.NewStyle().
			Background(infoColor).
			Foreground(textColor).
			Padding(0, 1)

	InfoText = lipgloss.NewStyle().
			Foreground(infoColor)

	SuccessBlock = lipgloss.NewStyle().
			Background(successColor).
			Foreground(textColor).
			Padding(0, 1)

	SuccessText = lipgloss.NewStyle().
			Foreground(successColor)

	ErrorBlock = lipgloss.NewStyle().
			Background(errorColor).
			Foreground(textColor).
			Padding(0, 1)

	ErrorText = lipgloss.NewStyle().
			Foreground(errorColor)

	DebugBlock = lipgloss.NewStyle().
			Background(debugColor).
			Foreground(textColor).
			Padding(0, 1)

	DebugText = lipgloss.NewStyle().
			Foreground(debugColor)
)
