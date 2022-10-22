package log

import (
	"io"

	"github.com/pterm/pterm"
)

type LoggerOption = func(logger *PrefixedLogger)

func WithOutput(writer io.Writer) LoggerOption {
	return func(logger *PrefixedLogger) {
		logger.writer.Writer = writer
	}
}

func WithStyle(style *pterm.Style) LoggerOption {
	return func(logger *PrefixedLogger) {
		logger.writer.Prefix.Style = style
	}
}
