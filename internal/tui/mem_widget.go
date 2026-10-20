package tui

import (
	"fmt"
	"runtime"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/evg4b/uncors/internal/helpers"
)

var memWidgetStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#777777"))

type memUpdateMsg struct{ mb float64 }

func getMemUsage() float64 {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return float64(memStats.Sys) / bytesPerMegabyte
}

type MemoryWidget struct {
	memMB       float64
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
	widget.memMB = widget.getMemUsage()

	return widget
}

func (m *MemoryWidget) Init() tea.Cmd {
	return m.memTickCmd()
}

func (m *MemoryWidget) Update(msg tea.Msg) (*MemoryWidget, tea.Cmd) {
	if typedMsg, ok := msg.(memUpdateMsg); ok {
		m.memMB = typedMsg.mb

		return m, m.memTickCmd()
	}

	return m, nil
}

func (m *MemoryWidget) View() tea.View {
	content := memWidgetStyle.Render(fmt.Sprintf("[ %.1f MB ]", m.memMB))

	return tea.NewView(content)
}

func (m *MemoryWidget) memTickCmd() tea.Cmd {
	return tea.Tick(memTickInterval, func(_ time.Time) tea.Msg {
		return memUpdateMsg{
			mb: m.getMemUsage(),
		}
	})
}
