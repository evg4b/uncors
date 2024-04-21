package styles

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"math"
)

var noLevel = log.Level(math.MaxInt32)

var (
	boxLength = 8

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
			noLevel: lipgloss.NewStyle().Margin(0).Padding(0),
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

func CreateLogger(logger *log.Logger, prefix string) *log.Logger {
	newStyles := log.Styles{
		Timestamp: DefaultStyles.Timestamp.Copy(),
		Caller:    DefaultStyles.Caller.Copy(),
		Prefix:    DefaultStyles.Prefix.Copy(),
		Message:   DefaultStyles.Message.Copy(),
		Key:       DefaultStyles.Key.Copy(),
		Value:     DefaultStyles.Value.Copy(),
		Separator: DefaultStyles.Separator.Copy(),
		Levels:    make(map[log.Level]lipgloss.Style, len(DefaultStyles.Levels)),
		Keys:      make(map[string]lipgloss.Style, len(DefaultStyles.Keys)),
		Values:    make(map[string]lipgloss.Style, len(DefaultStyles.Values)),
	}

	for level, style := range DefaultStyles.Levels {
		if level == noLevel {
			newStyles.Levels[level] = style.Copy().
				SetString(prefix + style.Value())
		} else {
			newStyles.Levels[level] = style.Copy().
				SetString(prefix, style.Value())
		}
	}

	copyMap(DefaultStyles.Keys, newStyles.Keys)
	copyMap(DefaultStyles.Values, newStyles.Values)

	newLogger := logger.With()
	newLogger.SetStyles(&newStyles)

	return newLogger
}

func copyMap(source, dest map[string]lipgloss.Style) {
	for key, value := range source {
		dest[key] = value.Copy()
	}
}
