package log

import (
	"os"

	"github.com/pterm/pterm"
)

type PrefixedLogger struct {
	writer *pterm.PrefixPrinter
}

func NewLogger(name string, options ...LoggerOption) *PrefixedLogger {
	logger := &PrefixedLogger{
		writer: &pterm.PrefixPrinter{
			Writer:       os.Stdout,
			MessageStyle: &pterm.ThemeDefault.DefaultText,
			Prefix: pterm.Prefix{
				Style: &pterm.ThemeDefault.DefaultText,
				Text:  name,
			},
		},
	}

	for _, option := range options {
		option(logger)
	}

	return logger
}

func (logger *PrefixedLogger) Error(v ...interface{}) {
	logger.writer.Println(errorPrinter.Sprint(v...))
}

func (logger *PrefixedLogger) Errorf(template string, v ...interface{}) {
	logger.writer.Println(errorPrinter.Sprintf(template, v...))
}

func (logger *PrefixedLogger) Warning(v ...interface{}) {
	logger.writer.Println(warningPrinter.Sprint(v...))
}

func (logger *PrefixedLogger) Warningf(template string, v ...interface{}) {
	logger.writer.Println(warningPrinter.Sprintf(template, v...))
}

func (logger *PrefixedLogger) Info(v ...interface{}) {
	logger.writer.Println(infoPrinter.Sprint(v...))
}

func (logger *PrefixedLogger) Infof(template string, v ...interface{}) {
	logger.writer.Println(infoPrinter.Sprintf(template, v...))
}

func (logger *PrefixedLogger) Debug(v ...interface{}) {
	if pterm.PrintDebugMessages {
		logger.writer.Println(debugPrinter.Sprint(v...))
	}
}

func (logger *PrefixedLogger) Debugf(template string, v ...interface{}) {
	if pterm.PrintDebugMessages {
		logger.writer.Println(debugPrinter.Sprintf(template, v...))
	}
}
