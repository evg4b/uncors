package tuiapp

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryWidget(t *testing.T) {
	t.Run("NewMemoryWidget initializes with valid memory", func(t *testing.T) {
		widget := NewMemoryWidget()
		assert.NotNil(t, widget)
		assert.GreaterOrEqual(t, widget.memMB, 0.0)
	})

	t.Run("WithMemoryUsage sets where the memory comes from", func(t *testing.T) {
		widget := NewMemoryWidget(WithMemoryUsage(func() float64 { return 7 }))

		assert.Contains(t, widget.View().Content, "[ 7.0 MB ]")
	})

	t.Run("Init returns memory tick command", func(t *testing.T) {
		widget := NewMemoryWidget()
		cmd := widget.Init()
		require.NotNil(t, cmd)
	})

	t.Run("Update handles memUpdateMsg", func(t *testing.T) {
		widget := NewMemoryWidget()
		widget.memMB = 10.0

		msg := memUpdateMsg{mb: 15.5}
		newWidget, cmd := widget.Update(msg)

		assert.InDelta(t, 15.5, newWidget.memMB, 0.0001)
		assert.Same(t, widget, newWidget)
		assert.NotNil(t, cmd)
	})

	t.Run("Update ignores other messages", func(t *testing.T) {
		widget := NewMemoryWidget()
		widget.memMB = 10.0

		msg := tea.KeyPressMsg(tea.Key{Text: "a", Code: 'a'})
		newWidget, cmd := widget.Update(msg)

		assert.InDelta(t, 10.0, newWidget.memMB, 0.0001)
		assert.Same(t, widget, newWidget)
		assert.Nil(t, cmd)
	})

	t.Run("View renders correctly", func(t *testing.T) {
		widget := NewMemoryWidget()
		widget.memMB = 12.34

		view := widget.View()
		assert.Contains(t, view.Content, "[ 12.3 MB ]")
	})

	t.Run("memTickCmd produces memUpdateMsg", func(t *testing.T) {
		cmd := NewMemoryWidget().memTickCmd()
		msg := cmd()

		assert.NotNil(t, msg)
	})
}
