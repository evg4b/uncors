package tui

import (
	"math"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/evg4b/uncors/internal/tui/styles"
)

var noLevel = log.Level(math.MaxInt32)

var (
	boxLength = 8

	debugPrefix = styles.DebugBlockStyle.Width(boxLength)
	infoPrefix  = styles.InfoBlockStyle.Width(boxLength)
	warnPrefix  = styles.WarningBlockStyle.Width(boxLength)
	errorPrefix = styles.ErrorBlockStyle.Width(boxLength)

	defaultStyle = lipgloss.NewStyle()

	DefaultStyles = log.Styles{
		Timestamp: defaultStyle,
		Caller:    defaultStyle.Faint(true),
		Prefix:    defaultStyle.Bold(true).Faint(true),
		Message:   defaultStyle,
		Key:       defaultStyle.Faint(true),
		Value:     defaultStyle.Faint(true),
		Separator: defaultStyle.Faint(true),
		Levels: map[log.Level]lipgloss.Style{
			noLevel: defaultStyle.
				Margin(0).
				Padding(0),
			log.DebugLevel: defaultStyle.
				SetString(debugPrefix.Render(debugLabel)).
				Bold(true),
			log.InfoLevel: defaultStyle.
				SetString(infoPrefix.Render(infoLabel)).
				Bold(true),
			log.WarnLevel: defaultStyle.
				SetString(warnPrefix.Render(warningLabel)).
				Bold(true),
			log.ErrorLevel: defaultStyle.
				SetString(errorPrefix.Render(errorLabel)).
				Bold(true),
			log.FatalLevel: defaultStyle.
				SetString(errorPrefix.Render(fatalLabel)).
				Bold(true),
		},
		Keys:   map[string]lipgloss.Style{},
		Values: map[string]lipgloss.Style{},
	}
)

func CreateLogger(logger *log.Logger, prefix string) *log.Logger {
	newStyles := &log.Styles{
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
	newLogger.SetStyles(newStyles)

	return newLogger
}

func copyMap(source, dest map[string]lipgloss.Style) {
	for key, value := range source {
		dest[key] = value
	}
}
