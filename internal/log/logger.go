package log

import (
	"net/http"

	"github.com/pterm/pterm"
)

type PrefixedLogger struct {
	writer *pterm.PrefixPrinter
	debug  *pterm.PrefixPrinter
}

func NewLogger(name string, options ...LoggerOption) *PrefixedLogger {
	logger := &PrefixedLogger{
		writer: &pterm.PrefixPrinter{
			MessageStyle: &pterm.ThemeDefault.DefaultText,
			Prefix: pterm.Prefix{
				Style: &pterm.ThemeDefault.DefaultText,
				Text:  name,
			},
		},
		debug: &pterm.PrefixPrinter{
			MessageStyle: &pterm.ThemeDefault.DebugMessageStyle,
			Prefix: pterm.Prefix{
				Text:  name,
				Style: &pterm.ThemeDefault.DebugPrefixStyle,
			},
			Debugger: true,
		},
	}

	for _, option := range options {
		option(logger)
	}

	return logger
}

func (logger *PrefixedLogger) Error(v ...any) {
	logger.writer.Println(errorPrinter.Sprint(v...))
}

func (logger *PrefixedLogger) Errorf(template string, v ...any) {
	logger.writer.Println(errorPrinter.Sprintf(template, v...))
}

func (logger *PrefixedLogger) Warning(v ...any) {
	logger.writer.Println(warningPrinter.Sprint(v...))
}

func (logger *PrefixedLogger) Warningf(template string, v ...any) {
	logger.writer.Println(warningPrinter.Sprintf(template, v...))
}

func (logger *PrefixedLogger) Info(v ...any) {
	logger.writer.Println(infoPrinter.Sprint(v...))
}

func (logger *PrefixedLogger) Infof(template string, v ...any) {
	logger.writer.Println(infoPrinter.Sprintf(template, v...))
}

func (logger *PrefixedLogger) Debug(v ...any) {
	if pterm.PrintDebugMessages {
		logger.debug.Println(debugPrinter.Sprint(v...))
	}
}

func (logger *PrefixedLogger) Debugf(template string, v ...any) {
	if pterm.PrintDebugMessages {
		logger.debug.Println(debugPrinter.Sprintf(template, v...))
	}
}

func (logger *PrefixedLogger) PrintResponse(response *http.Response) {
	logger.writer.Println(printResponse(response))
}
