package styles

import "github.com/charmbracelet/lipgloss"

// TODO: Replace to adaptive colors.
var (
	LogoYellow = lipgloss.NewStyle().
			Foreground(yellow)

	LogoRed = lipgloss.NewStyle().
		Foreground(red)

	WarningBlock = lipgloss.NewStyle().
			Background(yellow).
			Foreground(black).
			Padding(0, 1).
			ColorWhitespace(true)

	WarningText = lipgloss.NewStyle().
			Foreground(yellow)

	InfoBlock = lipgloss.NewStyle().
			Background(blue).
			Foreground(black).
			Padding(0, 1).
			ColorWhitespace(true)

	InfoText = lipgloss.NewStyle().
			Foreground(blue)

	SuccessBlock = lipgloss.NewStyle().
			Background(green).
			Foreground(black).
			Padding(0, 1).
			ColorWhitespace(true)

	SuccessText = lipgloss.NewStyle().
			Foreground(green).
			ColorWhitespace(true)

	ErrorBlock = lipgloss.NewStyle().
			Background(red).
			Foreground(black).
			Padding(0, 1).
			ColorWhitespace(true)

	ErrorText = lipgloss.NewStyle().
			Foreground(red)

	DebugBlock = lipgloss.NewStyle().
			Background(grey).
			Foreground(black).
			Padding(0, 1).
			ColorWhitespace(true)

	DebugText = lipgloss.NewStyle().
			Foreground(grey)

	DisabledBlock = lipgloss.NewStyle().
			Background(darkGrey).
			Foreground(black).
			Padding(0, 1).
			ColorWhitespace(true)

	DisabledText = lipgloss.NewStyle().
			Foreground(darkGrey).
			ColorWhitespace(true)
)
