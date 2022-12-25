package log

import (
	"fmt"
	"io"
	"os"

	"github.com/pterm/pterm"
)

func Fatal(a ...any) {
	Error(a...)
	os.Exit(0)
}

func Error(a ...any) {
	errorPrinter.Println(a...)
}

func Errorf(template string, a ...any) {
	Error(fmt.Sprintf(template, a...))
}

func Warning(a ...any) {
	warningPrinter.Println(a...)
}

func Warningf(template string, a ...any) {
	Warning(fmt.Sprintf(template, a...))
}

func Info(a ...any) {
	infoPrinter.Println(a...)
}

func Infof(template string, a ...any) {
	Info(fmt.Sprintf(template, a...))
}

func Debug(a ...any) {
	debugPrinter.Println(a...)
}

func Debugf(template string, a ...any) {
	Debug(fmt.Sprintf(template, a...))
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
