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

	InfoBlock = lipgloss.NewStyle().
			Background(lipgloss.Color(blue)).
			Foreground(lipgloss.Color(black)).
			Padding(0, 1).
			Margin(0, 1, 0, 0).
			ColorWhitespace(true)

	InfoText = lipgloss.NewStyle().
			Foreground(lipgloss.Color(blue))

	SuccessBlock = lipgloss.NewStyle().
			Background(lipgloss.Color(green)).
			Foreground(lipgloss.Color(black)).
			Padding(0, 1).
			Margin(0, 1, 0, 0).
			ColorWhitespace(true)

	SuccessText = lipgloss.NewStyle().
			Foreground(lipgloss.Color(green)).
			ColorWhitespace(true)

	ErrorBlock = lipgloss.NewStyle().
			Background(lipgloss.Color(red)).
			Foreground(lipgloss.Color(black)).
			Padding(0, 1).
			Margin(0, 1, 0, 0).
			ColorWhitespace(true)

	ErrorText = lipgloss.NewStyle().
			Foreground(lipgloss.Color(red))

	DebugBlock = lipgloss.NewStyle().
			Background(lipgloss.Color(grey)).
			Foreground(lipgloss.Color(black)).
			Padding(0, 1).
			Margin(0, 1, 0, 0).
			ColorWhitespace(true)

	DebugText = lipgloss.NewStyle().
			Foreground(lipgloss.Color(grey))

	DisabledBlock = lipgloss.NewStyle().
			Background(lipgloss.Color(darkGrey)).
			Foreground(lipgloss.Color(black)).
			Padding(0, 1).
			Margin(0, 1, 0, 0).
			ColorWhitespace(true)

	DisabledText = lipgloss.NewStyle().
			Foreground(lipgloss.Color(darkGrey)).
			ColorWhitespace(true)
)
