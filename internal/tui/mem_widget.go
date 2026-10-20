package tui

import (
	"fmt"
	"runtime"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/tui/styles"
)

type memUpdateMsg struct{ mb float64 }

func getMemUsage() float64 {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return float64(memStats.Sys) / bytesPerMegabyte
}

type MemoryWidget struct {
	memMB       float64
	rendered    string
	getMemUsage func() float64
}

type MemoryWidgetOption = func(*MemoryWidget)

func WithMemoryUsage(usage func() float64) MemoryWidgetOption {
	return func(widget *MemoryWidget) {
		widget.getMemUsage = usage
	}
}

func NewMemoryWidget(options ...MemoryWidgetOption) *MemoryWidget {
	widget := helpers.ApplyOptions(&MemoryWidget{getMemUsage: getMemUsage}, options)
	widget.setUsage(widget.getMemUsage())

	return widget
}

func (m *MemoryWidget) Init() tea.Cmd {
	return m.memTickCmd()
}

func (m *MemoryWidget) Update(msg tea.Msg) (*MemoryWidget, tea.Cmd) {
	if typedMsg, ok := msg.(memUpdateMsg); ok {
		m.setUsage(typedMsg.mb)

		return m, m.memTickCmd()
	}

	return m, nil
}

func (m *MemoryWidget) View() tea.View {
	return tea.NewView(m.rendered)
}

func (m *MemoryWidget) setUsage(memMB float64) {
	if memMB == m.memMB && m.rendered != "" {
		return
	}

	m.memMB = memMB
	m.rendered = styles.Default.MutedText.Render(fmt.Sprintf("[ %.1f MB ]", memMB))
}

func (m *MemoryWidget) memTickCmd() tea.Cmd {
	return tea.Tick(memTickInterval, func(_ time.Time) tea.Msg {
		return memUpdateMsg{
			mb: m.getMemUsage(),
		}
	})
}
