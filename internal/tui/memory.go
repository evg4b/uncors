package tui

import (
	"runtime"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dustin/go-humanize"
)

type MemoryTick string

type memoryTracker struct {
	usage    string
	memStats *runtime.MemStats
	ticker   *time.Ticker
}

func NewMemoryTracker() tea.Model {
	return memoryTracker{
		usage:    "...",
		memStats: &runtime.MemStats{},
		ticker:   time.NewTicker(time.Second),
	}
}

func (m memoryTracker) Init() tea.Cmd {
	return tea.Sequence(m.mem, m.tick)
}

func (m memoryTracker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if usage, ok := msg.(MemoryTick); ok {
		m.usage = string(usage)

		return m, m.tick
	}

	return m, nil
}

func (m memoryTracker) tick() tea.Msg {
	<-m.ticker.C

	return m.mem()
}

func (m memoryTracker) View() string {
	return m.usage
}

func (m memoryTracker) mem() tea.Msg {
	runtime.ReadMemStats(m.memStats)

	return MemoryTick(humanize.Bytes(m.memStats.Alloc))
}
