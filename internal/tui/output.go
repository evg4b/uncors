package tui

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/tui/styles"
)

type ouputType int8

const (
	defaultOutput ouputType = iota
	infoOutput
	warnOutput
	errorOutput
)

var boxLength = 8

var levelStyles = map[ouputType]lipgloss.Style{
	infoOutput:    styles.InfoBlockStyle.Width(boxLength).Bold(true),
	warnOutput:    styles.WarningBlockStyle.Width(boxLength).Bold(true),
	errorOutput:   styles.ErrorBlockStyle.Width(boxLength).Bold(true),
	defaultOutput: lipgloss.NewStyle(),
}

var messageMap = map[ouputType]string{
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

func NewCliOutput(output io.Writer, options ...Option) *CliOutput {
	return helpers.ApplyOptions(&CliOutput{
		mutex:  &sync.RWMutex{},
		output: output,
		buffer: bytes.Buffer{},
	}, options)
}

func (o *CliOutput) Write(p []byte) (int, error) {
	o.mutex.RLock()
	defer o.mutex.RUnlock()

	return o.output.Write(p)
}

func (o *CliOutput) Info(msg any) {
	o.print(fmt.Sprint(msg), infoOutput)
}

func (o *CliOutput) Error(msg any) {
	o.print(fmt.Sprint(msg), errorOutput)
}

func (o *CliOutput) Errorf(msg string, args ...any) {
	o.print(fmt.Sprintf(msg, args...), errorOutput)
}

func (o *CliOutput) Print(msg any) {
	o.print(fmt.Sprint(msg), defaultOutput)
}

func (o *CliOutput) Request(data *contracts.ReqestData) {
	o.print(printResponse(data), defaultOutput)
}

func (o *CliOutput) print(msg string, outputType ouputType) {
	o.renderPrefix()
	o.renderLevel(outputType)
	o.renderMessage(msg)
	o.buffer.WriteByte('\n')

	err := o.flushBuffer()
	if err != nil {
		panic(err)
	}
}

func (o *CliOutput) renderLevel(level ouputType) {
	renderer := levelStyles[level]
	if levelMessage, ok := messageMap[level]; ok {
		o.buffer.WriteString(renderer.Render(levelMessage))
	}

	o.buffer.WriteByte(' ')
}

func (o *CliOutput) renderMessage(msg string) {
	msg = strings.TrimSuffix(msg, "\n")
	fmt.Fprint(&o.buffer, msg)
}

func (o *CliOutput) renderPrefix() {
	if len(o.prefix) > 0 {
		o.buffer.WriteString(o.prefix)
	}
}

func (o *CliOutput) flushBuffer() error {
	if o.buffer.Len() == 0 {
		return nil
	}

	o.mutex.Lock()
	defer o.mutex.Unlock()
	defer o.buffer.Reset()

	_, err := o.output.Write(o.buffer.Bytes())

	return err
}
