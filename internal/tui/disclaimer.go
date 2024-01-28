package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var warningStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("#FFD400")).
	Foreground(lipgloss.Color("#000")).
	Strikethrough(true).
	Padding(0, 1).
	Strikethrough(true).
	Margin(0, 1, 0, 0).
	ColorWhitespace(true)

var PrintDisclaimerMessage = tea.Println(lipgloss.JoinHorizontal(
	lipgloss.Left,
	warningStyle.Render("Warning\n\n"),
	DisclaimerMessage,
))
