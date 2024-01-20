package tui

import (
	"io"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/evg4b/linebyline"
)

const logBufferSize = 100

type PrinterMsg []byte

type Printer struct {
	data   chan []byte
	output io.WriteCloser
}

func NewPrinter() Printer {
	dataChanel := make(chan []byte, logBufferSize)

	return Printer{
		data: dataChanel,
		output: linebyline.NewByLineWriter(
			linebyline.OmitNewLineRune(),
			linebyline.WithFlushFunc(func(bytes []byte) error {
				dataChanel <- bytes

				return nil
			}),
		),
	}
}

func (p Printer) Tick() tea.Msg {
	return PrinterMsg(<-p.data)
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
	close(p.data)

	return p.output.Close()
}