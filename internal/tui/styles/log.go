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
			Padding(0, 1)

	WarningText = lipgloss.NewStyle().
			Foreground(yellow)

	InfoBlock = lipgloss.NewStyle().
			Background(blue).
			Foreground(black).
			Padding(0, 1)

	InfoText = lipgloss.NewStyle().
			Foreground(blue)

	SuccessBlock = lipgloss.NewStyle().
			Background(green).
			Foreground(black).
			Padding(0, 1)

	SuccessText = lipgloss.NewStyle().
			Foreground(green)

	ErrorBlock = lipgloss.NewStyle().
			Background(red).
			Foreground(black).
			Padding(0, 1)

	ErrorText = lipgloss.NewStyle().
			Foreground(red)

	DebugBlock = lipgloss.NewStyle().
			Background(grey).
			Foreground(black).
			Padding(0, 1)

	DebugText = lipgloss.NewStyle().
			Foreground(grey)

	DisabledBlock = lipgloss.NewStyle().
			Background(darkGrey).
			Foreground(black).
			Padding(0, 1)

	DisabledText = lipgloss.NewStyle().
			Foreground(darkGrey)
)
