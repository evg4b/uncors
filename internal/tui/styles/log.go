package styles

import "github.com/charmbracelet/lipgloss"

// TODO: Replace to adaptive colors.
var (
	LogoYellow = lipgloss.NewStyle().
			Foreground(lipgloss.Color(yellow))

	LogoRed = lipgloss.NewStyle().
		Foreground(lipgloss.Color(red))

	WarningBlock = lipgloss.NewStyle().
			Background(lipgloss.Color(yellow)).
			Foreground(lipgloss.Color(black)).
			Padding(0, 1).
			Margin(0, 1, 0, 0).
			ColorWhitespace(true)

	WarningText = lipgloss.NewStyle().
			Foreground(lipgloss.Color(yellow))

	SuccessBlock = lipgloss.NewStyle().
			Background(lipgloss.Color(green)).
			Foreground(lipgloss.Color(black)).
			Padding(0, 1).
			Margin(0, 1, 0, 0).
			ColorWhitespace(true)

	SuccessText = lipgloss.NewStyle().
			Foreground(lipgloss.Color(green)).
			Padding(0, 1).
			Margin(0, 1, 0, 0).
			ColorWhitespace(true)

	ErrorBlock = lipgloss.NewStyle().
			Background(lipgloss.Color(red)).
			Foreground(lipgloss.Color(black)).
			Padding(0, 1).
			Margin(0, 1, 0, 0).
			ColorWhitespace(true)

	ErrorText = lipgloss.NewStyle().
			Foreground(lipgloss.Color(red))

	DisabledText = lipgloss.NewStyle().
			Foreground(lipgloss.Color(grey)).
			Padding(0, 1).
			Margin(0, 1, 0, 0).
			ColorWhitespace(true)

	DisabledBlock = lipgloss.NewStyle().
			Background(lipgloss.Color(grey)).
			Foreground(lipgloss.Color(black)).
			Padding(0, 1).
			Margin(0, 1, 0, 0).
			ColorWhitespace(true)
)
