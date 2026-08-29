package uncorsapp

import (
	"bytes"
	"strings"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/tui"
)

// Output is the TUI's contracts.Output implementation: every line it is given
// is queued on a channel that the model drains and renders. It is created before
// the container so that the container can be built with it, instead of having it
// swapped in after other components may already have captured the default one.
type Output struct {
	ch     chan string
	prefix string
}

func NewOutput() *Output {
	return &Output{ch: make(chan string, outputChannelSize)}
}

// Lines returns the channel of rendered output lines for the model to drain.
func (o *Output) Lines() <-chan string {
	return o.ch
}

func (o *Output) Write(p []byte) (int, error) {
	o.send(string(p))

	return len(p), nil
}

func (o *Output) Info(msg any) {
	o.capture(func(out *tui.CliOutput) { out.Info(msg) })
}

func (o *Output) Infof(msg string, args ...any) {
	o.capture(func(out *tui.CliOutput) { out.Infof(msg, args...) })
}

func (o *Output) InfoBox(messages ...string) {
	o.captureBox(func(out *tui.CliOutput) { out.InfoBox(messages...) })
}

func (o *Output) Error(msg any) {
	o.capture(func(out *tui.CliOutput) { out.Error(msg) })
}

func (o *Output) Errorf(msg string, args ...any) {
	o.capture(func(out *tui.CliOutput) { out.Errorf(msg, args...) })
}

func (o *Output) ErrorBox(messages ...string) {
	o.captureBox(func(out *tui.CliOutput) { out.ErrorBox(messages...) })
}

func (o *Output) Warn(msg any) {
	o.capture(func(out *tui.CliOutput) { out.Warn(msg) })
}

func (o *Output) Warnf(msg string, args ...any) {
	o.capture(func(out *tui.CliOutput) { out.Warnf(msg, args...) })
}

func (o *Output) WarnBox(messages ...string) {
	o.captureBox(func(out *tui.CliOutput) { out.WarnBox(messages...) })
}

func (o *Output) Print(msg any) {
	o.capture(func(out *tui.CliOutput) { out.Print(msg) })
}

func (o *Output) Printf(msg string, args ...any) {
	o.capture(func(out *tui.CliOutput) { out.Printf(msg, args...) })
}

func (o *Output) Request(data *contracts.RequestData) {
	o.capture(func(out *tui.CliOutput) { out.Request(data) })
}

func (o *Output) NewPrefixOutput(prefix string) contracts.Output {
	return &Output{
		ch:     o.ch,
		prefix: prefix,
	}
}

func (o *Output) send(msg string) {
	msg = strings.TrimRight(msg, "\n")
	if len(msg) > 0 {
		select {
		case o.ch <- msg:
		default:
		}
	}
}

func (o *Output) capture(fn func(out *tui.CliOutput)) {
	var buf bytes.Buffer

	tmp := tui.NewCliOutput(&buf, tui.WithPrefix(o.prefix))
	fn(tmp)
	o.send(buf.String())
}

func (o *Output) captureBox(fn func(out *tui.CliOutput)) {
	var buf bytes.Buffer
	fn(tui.NewCliOutput(&buf))
	o.send(buf.String())
}
