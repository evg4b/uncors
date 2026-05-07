package uncorsapp

import (
	"bytes"
	"strings"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/tui"
)

type tuiOutput struct {
	ch     chan<- string
	prefix string
}

func newTuiOutput(ch chan<- string) *tuiOutput {
	return &tuiOutput{ch: ch}
}

func (o *tuiOutput) Write(p []byte) (int, error) {
	o.send(string(p))

	return len(p), nil
}

func (o *tuiOutput) Info(msg any) {
	o.capture(func(out *tui.CliOutput) { out.Info(msg) })
}

func (o *tuiOutput) Infof(msg string, args ...any) {
	o.capture(func(out *tui.CliOutput) { out.Infof(msg, args...) })
}

func (o *tuiOutput) InfoBox(messages ...string) {
	o.captureBox(func(out *tui.CliOutput) { out.InfoBox(messages...) })
}

func (o *tuiOutput) Error(msg any) {
	o.capture(func(out *tui.CliOutput) { out.Error(msg) })
}

func (o *tuiOutput) Errorf(msg string, args ...any) {
	o.capture(func(out *tui.CliOutput) { out.Errorf(msg, args...) })
}

func (o *tuiOutput) ErrorBox(messages ...string) {
	o.captureBox(func(out *tui.CliOutput) { out.ErrorBox(messages...) })
}

func (o *tuiOutput) Warn(msg any) {
	o.capture(func(out *tui.CliOutput) { out.Warn(msg) })
}

func (o *tuiOutput) Warnf(msg string, args ...any) {
	o.capture(func(out *tui.CliOutput) { out.Warnf(msg, args...) })
}

func (o *tuiOutput) WarnBox(messages ...string) {
	o.captureBox(func(out *tui.CliOutput) { out.WarnBox(messages...) })
}

func (o *tuiOutput) Print(msg any) {
	o.capture(func(out *tui.CliOutput) { out.Print(msg) })
}

func (o *tuiOutput) Printf(msg string, args ...any) {
	o.capture(func(out *tui.CliOutput) { out.Printf(msg, args...) })
}

func (o *tuiOutput) Request(data *contracts.ReqestData) {
	o.capture(func(out *tui.CliOutput) { out.Request(data) })
}

func (o *tuiOutput) NewPrefixOutput(prefix string) contracts.Output {
	return &tuiOutput{
		ch:     o.ch,
		prefix: prefix,
	}
}

func (o *tuiOutput) send(msg string) {
	msg = strings.TrimRight(msg, "\n")
	if len(msg) > 0 {
		select {
		case o.ch <- msg:
		default:
		}
	}
}

func (o *tuiOutput) capture(fn func(out *tui.CliOutput)) {
	var buf bytes.Buffer

	tmp := tui.NewCliOutput(&buf, tui.WithPrefix(o.prefix))
	fn(tmp)
	o.send(buf.String())
}

func (o *tuiOutput) captureBox(fn func(out *tui.CliOutput)) {
	var buf bytes.Buffer
	fn(tui.NewCliOutput(&buf))
	o.send(buf.String())
}
