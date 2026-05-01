package styles

import lipgloss "charm.land/lipgloss/v2"

var underlineStyle = lipgloss.NewStyle().
	Underline(true)

var paddedStyle = lipgloss.NewStyle().
	Padding(0, 1)

const baseBlockWidth = 8

var blockStyle = lipgloss.NewStyle().
	Foreground(contrastColor).
	Padding(0, 1).
	Margin(0).
	Width(baseBlockWidth)
