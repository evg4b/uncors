package log

import (
	"fmt"
	"io"
	"os"

	"github.com/pterm/pterm"
)

type Logger interface {
	Error(a ...interface{})
	Errorf(template string, a ...interface{})
	Warning(a ...interface{})
	Warningf(template string, a ...interface{})
	Info(a ...interface{})
	Infof(template string, a ...interface{})
	Debug(a ...interface{})
	Debugf(template string, a ...interface{})
}

func Fatal(a ...interface{}) {
	pterm.Error.Println(a...)
	os.Exit(0)
}

func Error(a ...interface{}) {
	pterm.Error.Println(a...)
}

func Errorf(template string, a ...interface{}) {
	pterm.Error.Println(fmt.Sprintf(template, a...))
}

func Warning(a ...interface{}) {
	pterm.Warning.Println(a...)
}

func Warningf(template string, a ...interface{}) {
	pterm.Warning.Println(fmt.Sprintf(template, a...))
}

func Info(a ...interface{}) {
	pterm.Info.Println(a...)
}

func Infof(template string, a ...interface{}) {
	pterm.Info.Println(fmt.Sprintf(template, a...))
}

func Debug(a ...interface{}) {
	pterm.Debug.Println(a...)
}

func Debugf(template string, a ...interface{}) {
	pterm.Debug.Println(fmt.Sprintf(template, a...))
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

func SetLogger(output io.Writer) {
	pterm.SetDefaultOutput(output)
}
