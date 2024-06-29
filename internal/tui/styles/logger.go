package styles

import (
	"math"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
)

var noLevel = log.Level(math.MaxInt32)

var (
	boxLength = 8

	debugPrefix = DebugBlock.Width(boxLength)
	infoPrefix  = InfoBlock.Width(boxLength)
	warnPrefix  = WarningBlock.Width(boxLength)
	errorPrefix = ErrorBlock.Width(boxLength)

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
			log.DebugLevel: DebugText.
				SetString(debugPrefix.Render(DebugLabel)).
				Bold(true),
			log.InfoLevel: InfoText.
				SetString(infoPrefix.Render(InfoLabel)).
				Bold(true),
			log.WarnLevel: WarningText.
				SetString(warnPrefix.Render(WarningLabel)).
				Bold(true),
			log.ErrorLevel: ErrorText.
				SetString(errorPrefix.Render(ErrorLabel)).
				Bold(true),
			log.FatalLevel: ErrorText.
				SetString(errorPrefix.Render(FatalLabel)).
				Bold(true),
		},
		Keys:   map[string]lipgloss.Style{},
		Values: map[string]lipgloss.Style{},
	}
)

func CreateLogger(logger *log.Logger, prefix string) *log.Logger {
	newStyles := log.Styles{
		Timestamp: DefaultStyles.Timestamp,
		Caller:    DefaultStyles.Caller,
		Prefix:    DefaultStyles.Prefix,
		Message:   DefaultStyles.Message,
		Key:       DefaultStyles.Key,
		Value:     DefaultStyles.Value,
		Separator: DefaultStyles.Separator,
		Levels:    make(map[log.Level]lipgloss.Style, len(DefaultStyles.Levels)),
		Keys:      make(map[string]lipgloss.Style, len(DefaultStyles.Keys)),
		Values:    make(map[string]lipgloss.Style, len(DefaultStyles.Values)),
	}

	for level, style := range DefaultStyles.Levels {
		if level == noLevel {
			newStyles.Levels[level] = style.SetString(prefix + style.Value())
		} else {
			newStyles.Levels[level] = style.SetString(prefix, style.Value())
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
		dest[key] = value
	}
}
