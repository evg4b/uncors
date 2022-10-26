package log

import (
	"fmt"
	"io"
	"os"

	"github.com/pterm/pterm"
)

func Fatal(a ...interface{}) {
	Error(a...)
	os.Exit(0)
}

func Error(a ...interface{}) {
	errorPrinter.Println(a...)
}

func Errorf(template string, a ...interface{}) {
	Error(fmt.Sprintf(template, a...))
}

func Warning(a ...interface{}) {
	warningPrinter.Println(a...)
}

func Warningf(template string, a ...interface{}) {
	Warning(fmt.Sprintf(template, a...))
}

func Info(a ...interface{}) {
	infoPrinter.Println(a...)
}

func Infof(template string, a ...interface{}) {
	Info(fmt.Sprintf(template, a...))
}

func Debug(a ...interface{}) {
	debugPrinter.Println(a...)
}

func Debugf(template string, a ...interface{}) {
	Debug(fmt.Sprintf(template, a...))
}

func Print(a ...interface{}) {
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
