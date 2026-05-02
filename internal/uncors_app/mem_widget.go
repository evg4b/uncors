package uncorsapp

import (
	"fmt"
	"runtime"
	"time"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
)

var memWidgetStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#777777"))

type memUpdateMsg struct{ mb float64 }

type MemoryWidget struct {
	memMB float64
}

func NewMemoryWidget() *MemoryWidget {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return &MemoryWidget{
		memMB: float64(memStats.HeapAlloc) / bytesPerMegabyte,
	}
}

func (m *MemoryWidget) Init() tea.Cmd {
	return memTickCmd()
}

func (m *MemoryWidget) Update(msg tea.Msg) (*MemoryWidget, tea.Cmd) {
	if typedMsg, ok := msg.(memUpdateMsg); ok {
		m.memMB = typedMsg.mb

		return m, memTickCmd()
	}

	return m, nil
}

func (m *MemoryWidget) View() tea.View {
	content := memWidgetStyle.Render(fmt.Sprintf("[ %.1f MB ]", m.memMB))

	return tea.NewView(content)
}

func memTickCmd() tea.Cmd {
	return tea.Tick(memTickInterval, func(_ time.Time) tea.Msg {
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)

		return memUpdateMsg{mb: float64(memStats.HeapAlloc) / bytesPerMegabyte}
	})
}
