package tui

import (
	"io"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/evg4b/linebyline"
)

const logBufferSize = 100

type PrinterMsg []byte

type Printer struct {
	dataChanel chan []byte
	output     io.WriteCloser
}

func NewPrinter() Printer {
	dataChanel := make(chan []byte, logBufferSize)
	output := linebyline.NewByLineWriter(
		linebyline.OmitNewLineRune(),
		linebyline.WithFlushFunc(func(bytes []byte) error {
			dataChanel <- bytes

			return nil
		}),
	)

	return Printer{dataChanel, output}
}

func (p Printer) Tick() tea.Msg {
	return PrinterMsg(<-p.dataChanel)
}

func (p Printer) Update(msg tea.Msg) tea.Cmd {
	if msg, ok := msg.(PrinterMsg); ok {
		return tea.Batch(tea.Printf(string(msg)), p.Tick)
	}

	return nil
}

func (p Printer) Write(data []byte) (int, error) {
	return p.output.Write(data)
}

func (p Printer) Close() error {
	close(p.dataChanel)

	return p.output.Close()
}
