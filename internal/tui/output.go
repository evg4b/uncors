package tui

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/tui/styles"
)

type outputType int8

const (
	defaultOutput outputType = iota
	infoOutput
	warnOutput
	errorOutput
)

var boxLength = 8

var levelStyles = map[outputType]lipgloss.Style{
	infoOutput:    styles.InfoBlockStyle.Width(boxLength).Bold(true),
	warnOutput:    styles.WarningBlockStyle.Width(boxLength).Bold(true),
	errorOutput:   styles.ErrorBlockStyle.Width(boxLength).Bold(true),
	defaultOutput: lipgloss.NewStyle(),
}

var messageMap = map[outputType]string{
	infoOutput:  InfoLabel,
	warnOutput:  WarningLabel,
	errorOutput: ErrorLabel,
}

type CliOutput struct {
	mutex  *sync.RWMutex
	output io.Writer
	prefix string
	buffer bytes.Buffer
}

type Option = func(*CliOutput)

func WithPrefix(prefix string) Option {
	return func(o *CliOutput) {
		o.prefix = prefix
	}
}

func withMutex(mutex *sync.RWMutex) Option {
	return func(o *CliOutput) {
		o.mutex = mutex
	}
}

func NewCliOutput(output io.Writer, options ...Option) *CliOutput {
	return helpers.ApplyOptions(&CliOutput{
		mutex:  &sync.RWMutex{},
		output: output,
	}, options)
}

func (output *CliOutput) Write(p []byte) (int, error) {
	output.mutex.RLock()
	defer output.mutex.RUnlock()

	return output.output.Write(p)
}

func (output *CliOutput) Info(msg any) {
	output.print(fmt.Sprint(msg), infoOutput)
}

func (output *CliOutput) Infof(msg string, args ...any) {
	output.print(fmt.Sprintf(msg, args...), infoOutput)
}

func (output *CliOutput) InfoBox(messages ...string) {
	output.mutex.Lock()
	defer output.mutex.Unlock()

	output.printMessageBox(strings.Join(messages, "\n"), InfoLabel, styles.InfoBlockStyle)
}

func (output *CliOutput) Error(msg any) {
	output.print(fmt.Sprint(msg), errorOutput)
}

func (output *CliOutput) Errorf(msg string, args ...any) {
	output.print(fmt.Sprintf(msg, args...), errorOutput)
}

func (output *CliOutput) ErrorBox(messages ...string) {
	output.mutex.Lock()
	defer output.mutex.Unlock()

	output.printMessageBox(strings.Join(messages, "\n"), ErrorLabel, styles.ErrorBlockStyle)
}

func (output *CliOutput) Warn(msg any) {
	output.print(fmt.Sprint(msg), warnOutput)
}

func (output *CliOutput) Warnf(msg string, args ...any) {
	output.print(fmt.Sprintf(msg, args...), warnOutput)
}

func (output *CliOutput) WarnBox(messages ...string) {
	output.mutex.Lock()
	defer output.mutex.Unlock()

	output.printMessageBox(strings.Join(messages, "\n"), WarningLabel, styles.WarningBlockStyle)
}

func (output *CliOutput) Print(msg any) {
	output.print(fmt.Sprint(msg), defaultOutput)
}

func (output *CliOutput) Printf(msg string, args ...any) {
	output.print(fmt.Sprintf(msg, args...), defaultOutput)
}

func (output *CliOutput) Request(data *contracts.RequestData) {
	output.print(printResponse(data), defaultOutput)
}

func (output *CliOutput) NewPrefixOutput(prefix string) contracts.Output {
	return NewCliOutput(output.output, WithPrefix(prefix), withMutex(output.mutex))
}

// print holds the exclusive write lock for the full render+write cycle so that
// the shared buffer and the underlying writer are never accessed concurrently.
// Write() uses RLock only, which is blocked while print() owns the write lock.
func (output *CliOutput) print(msg string, level outputType) {
	output.mutex.Lock()
	defer output.mutex.Unlock()

	output.renderPrefix()
	output.renderLevel(level)
	output.renderMessage(msg)
	output.buffer.WriteByte('\n')

	_, err := output.output.Write(output.buffer.Bytes())
	output.buffer.Reset()

	if err != nil {
		panic(err)
	}
}

func (output *CliOutput) renderLevel(level outputType) {
	renderer := levelStyles[level]
	if levelMessage, ok := messageMap[level]; ok {
		output.buffer.WriteString(renderer.Render(levelMessage))
	}

	output.buffer.WriteByte(' ')
}

func (output *CliOutput) renderMessage(msg string) {
	msg = strings.TrimSuffix(msg, "\n")
	fmt.Fprint(&output.buffer, msg)
}

func (output *CliOutput) renderPrefix() {
	if len(output.prefix) > 0 {
		output.buffer.WriteString(output.prefix)
	}
}
