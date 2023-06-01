package log

import (
	"github.com/evg4b/uncors/internal/sfmt"
	"github.com/pterm/pterm"
	"io"
)

func Error(a ...any) {
	errorPrinter.Println(a...)
}

func Errorf(template string, a ...any) {
	Error(sfmt.Sprintf(template, a...))
}

func Warning(a ...any) {
	warningPrinter.Println(a...)
}

func Warningf(template string, a ...any) {
	Warning(sfmt.Sprintf(template, a...))
}

func Info(a ...any) {
	infoPrinter.Println(a...)
}

func Infof(template string, a ...any) {
	Info(sfmt.Sprintf(template, a...))
}

func Debug(a ...any) {
	debugPrinter.Println(a...)
}

func Debugf(template string, a ...any) {
	Debug(sfmt.Sprintf(template, a...))
}

func Print(a ...any) {
	pterm.Print(a...)
}

func EnableDebugMessages() {
	pterm.EnableDebugMessages()
}

func DisableDebugMessages() {
	pterm.DisableDebugMessages()
}

func DisableOutput() {
	pterm.DisableOutput()
}

func EnableOutput() {
	pterm.EnableOutput()
}

func DisableColor() {
	pterm.DisableColor()
}

func EnableColor() {
	pterm.EnableColor()
}

func SetOutput(output io.Writer) {
	pterm.SetDefaultOutput(output)
}
