package tuiapp

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
)

func TestHelpWidget(t *testing.T) {
	keys := NewKeyMap()

	t.Run("NewHelpWidget initializes correctly", func(t *testing.T) {
		widget := NewHelpWidget(keys)
		assert.NotNil(t, widget)
		assert.NotNil(t, widget.help)
		assert.False(t, widget.help.ShowAll)
		assert.Equal(t, 1, widget.Height())
		assert.Nil(t, widget.Init())
	})

	t.Run("Update handles tea.WindowSizeMsg", func(t *testing.T) {
		widget := NewHelpWidget(keys)

		msg := tea.WindowSizeMsg{Width: 100, Height: 50}
		newWidget, cmd := widget.Update(msg)

		assert.Same(t, widget, newWidget)
		assert.Nil(t, cmd)
	})

	t.Run("Update handles help toggle key", func(t *testing.T) {
		widget := NewHelpWidget(keys)

		msg := tea.KeyPressMsg(tea.Key{Text: "?", Code: '?'})
		newWidget, cmd := widget.Update(msg)

		assert.Same(t, widget, newWidget)
		assert.Nil(t, cmd)
		assert.True(t, widget.help.ShowAll)
		assert.Equal(t, 4, widget.Height())

		newWidget, cmd = widget.Update(msg)
		assert.Same(t, widget, newWidget)
		assert.Nil(t, cmd)
		assert.False(t, widget.help.ShowAll)
		assert.Equal(t, 1, widget.Height())
	})

	t.Run("Update ignores other keys", func(t *testing.T) {
		widget := NewHelpWidget(keys)

		msg := tea.KeyPressMsg(tea.Key{Text: "x", Code: 'x'})
		newWidget, cmd := widget.Update(msg)

		assert.Same(t, widget, newWidget)
		assert.Nil(t, cmd)
		assert.False(t, widget.help.ShowAll)
	})

	t.Run("View renders correctly", func(t *testing.T) {
		widget := NewHelpWidget(keys)

		view := widget.View()
		assert.NotEmpty(t, view.Content)
		assert.Contains(t, view.Content, "help")
	})
}
