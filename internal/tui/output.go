package tui

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
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

var faintStyle = lipgloss.NewStyle().Faint(true)

var messageMap = map[ouputType]string{
	infoOutput:  InfoLabel,
	warnOutput:  WarningLabel,
	errorOutput: ErrorLabel,
}

type CliOutput struct {
	mu     *sync.RWMutex
	output io.Writer
	prefix string
	buf    bytes.Buffer
}

func NewCliOutput(output io.Writer) *CliOutput {
	return &CliOutput{
		mu:     &sync.RWMutex{},
		output: output,
		buf:    bytes.Buffer{},
		prefix: "",
	}
}

func (o *CliOutput) Write(p []byte) (int, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	return o.output.Write(p)
}

func (o *CliOutput) Info(msg any) {
	panic("not implemented")
}

func (o *CliOutput) Error(msg any) {
	panic("not implemented")
}

func (o *CliOutput) Errorf(msg string, args ...any) {
	panic("not implemented")
}

func (o *CliOutput) print(msg string, outputType ouputType) {
	o.renderPrefix()
	o.renderLevel(outputType)
	o.renderMessage(msg)
	o.buf.WriteByte('\n')

	err := o.flushBuffer()
	if err != nil {
		panic(err)
	}
}

func (o *CliOutput) renderLevel(level ouputType) {
	renderer := levelStyles[level]
	if levelMessage, ok := messageMap[level]; ok {
		o.buf.WriteString(renderer.Render(levelMessage))
	}
	o.buf.WriteByte(' ')
}

func (o *CliOutput) renderMessage(msg string) {
	msg = strings.TrimSuffix(msg, "\n")
	fmt.Fprint(&o.buf, msg)
}

func (o *CliOutput) renderPrefix() {
	if len(o.prefix) > 0 {
		o.buf.WriteString(o.prefix)
	}
}

func (o *CliOutput) flushBuffer() error {
	if o.buf.Len() == 0 {
		return nil
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	defer o.buf.Reset()

	_, err := o.output.Write(o.buf.Bytes())

	return err
}
