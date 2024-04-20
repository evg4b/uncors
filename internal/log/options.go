package log

import (
	"github.com/pterm/pterm"
)

type LoggerOption = func(logger *PrefixedLogger)

func WithStyle(style *pterm.Style) LoggerOption {
	return func(logger *PrefixedLogger) {
		logger.writer.Prefix.Style = style
	}
}
