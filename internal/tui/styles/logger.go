package styles

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
)

// DefaultStyles returns the default styles.
func DefaultStyles() *log.Styles {
	return &log.Styles{
		Timestamp: lipgloss.NewStyle(),
		Caller:    lipgloss.NewStyle().Faint(true),
		Prefix:    lipgloss.NewStyle().Bold(true).Faint(true),
		Message:   lipgloss.NewStyle(),
		Key:       lipgloss.NewStyle().Faint(true),
		Value:     lipgloss.NewStyle(),
		Separator: lipgloss.NewStyle().Faint(true),
		Levels: map[log.Level]lipgloss.Style{
			log.DebugLevel: DebugText.Copy().
				SetString(DebugLabel).
				Bold(true),
			log.InfoLevel: InfoText.Copy().
				SetString(InfoLabel).
				Bold(true),
			log.WarnLevel: WarningText.Copy().
				SetString(WarningLabel).
				Bold(true),
			log.ErrorLevel: ErrorText.Copy().
				SetString(ErrorLabel).
				Bold(true),
			log.FatalLevel: ErrorText.Copy().
				SetString(FatalLabel).
				Bold(true),
		},
		Keys:   map[string]lipgloss.Style{},
		Values: map[string]lipgloss.Style{},
	}
}
