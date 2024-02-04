package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evg4b/uncors/internal/tui/styles"
)

var PrintDisclaimerMessage = tea.Println(lipgloss.JoinHorizontal(
	lipgloss.Bottom,
	styles.WarningBlock.Render("Warning"),
	DisclaimerMessage,
))
