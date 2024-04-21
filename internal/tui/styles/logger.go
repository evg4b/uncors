package styles

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
)

var (
	boxLength = 7

	debugPrefix = DebugBlock.Copy().Width(boxLength)
	infoPrefix  = InfoBlock.Copy().Width(boxLength)
	warnPrefix  = WarningBlock.Copy().Width(boxLength)
	errorPrefix = ErrorBlock.Copy().Width(boxLength)

	DefaultStyles = log.Styles{
		Timestamp: lipgloss.NewStyle(),
		Caller:    lipgloss.NewStyle().Faint(true),
		Prefix:    lipgloss.NewStyle().Bold(true).Faint(true),
		Message:   lipgloss.NewStyle(),
		Key:       lipgloss.NewStyle().Faint(true),
		Value:     lipgloss.NewStyle(),
		Separator: lipgloss.NewStyle().Faint(true),
		Levels: map[log.Level]lipgloss.Style{
			log.DebugLevel: DebugText.Copy().
				SetString(debugPrefix.Render(DebugLabel)).
				Bold(true),
			log.InfoLevel: InfoText.Copy().
				SetString(infoPrefix.Render(InfoLabel)).
				Bold(true),
			log.WarnLevel: WarningText.Copy().
				SetString(warnPrefix.Render(WarningLabel)).
				Bold(true),
			log.ErrorLevel: ErrorText.Copy().
				SetString(errorPrefix.Render(ErrorLabel)).
				Bold(true),
			log.FatalLevel: ErrorText.Copy().
				SetString(errorPrefix.Render(FatalLabel)).
				Bold(true),
		},
		Keys:   map[string]lipgloss.Style{},
		Values: map[string]lipgloss.Style{},
	}
)
