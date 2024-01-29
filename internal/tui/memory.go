package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dustin/go-humanize"
	"runtime"
	"time"
)

type MemoryTick string

type memoryTracker struct {
	usage  string
	ticker *time.Ticker
}

func NewMemoryTracker() tea.Model {
	return memoryTracker{
		usage:  "...",
		ticker: time.NewTicker(time.Second),
	}
}

func (m memoryTracker) Init() tea.Cmd {
	return tea.Sequence(
		m.mem,
		m.tick,
	)
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
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return MemoryTick(humanize.Bytes(memStats.Alloc))
}
