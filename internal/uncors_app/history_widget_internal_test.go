package uncorsapp

import (
	"os"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHistoryWidget(t *testing.T) {
	keys := newKeyMap()

	cleanup := func(widget *HistoryWidget) {
		if widget != nil && widget.hist != nil && widget.hist.file != nil {
			_, err := os.Stat(widget.hist.file.Name())
			if err == nil {
				_ = widget.Close()
			}
		}
	}

	t.Run("NewHistoryWidget initializes correctly", func(t *testing.T) {
		widget, err := NewHistoryWidget(keys)
		require.NoError(t, err)

		defer cleanup(widget)

		assert.NotNil(t, widget)
		assert.NotNil(t, widget.hist)
		assert.True(t, widget.autoScroll)
		assert.False(t, widget.HasLines())
		assert.Nil(t, widget.Init())
	})

	t.Run("Update handles tea.WindowSizeMsg", func(t *testing.T) {
		widget, err := NewHistoryWidget(keys)
		require.NoError(t, err)

		defer cleanup(widget)

		msg := tea.WindowSizeMsg{Width: 100, Height: 50}
		newWidget, cmd := widget.Update(msg)

		assert.Same(t, widget, newWidget)
		assert.Nil(t, cmd)
		assert.Equal(t, 100, widget.termWidth)
	})

	t.Run("Update handles outputLineMsg", func(t *testing.T) {
		widget, err := NewHistoryWidget(keys)
		require.NoError(t, err)

		defer cleanup(widget)

		msg := outputLineMsg("hello world")
		newWidget, cmd := widget.Update(msg)

		assert.Same(t, widget, newWidget)
		assert.Nil(t, cmd)
		assert.True(t, widget.HasLines())
		assert.Equal(t, 1, widget.hist.LineCount())
	})

	t.Run("Update handles restartMsg", func(t *testing.T) {
		widget, err := NewHistoryWidget(keys)
		require.NoError(t, err)

		defer cleanup(widget)

		newWidget, cmd := widget.Update(restartMsg{})
		assert.Same(t, widget, newWidget)
		assert.Nil(t, cmd)
	})

	t.Run("Update handles key presses", func(t *testing.T) {
		widget, err := NewHistoryWidget(keys)
		require.NoError(t, err)

		defer cleanup(widget)

		widget.termWidth = 80
		widget.vp.SetWidth(80)
		widget.SetHeight(5)

		// Add some lines so scrolling is possible
		for range 10 {
			_, _ = widget.Update(outputLineMsg("line"))
		}

		widget.autoScroll = true

		// Home key -> GotoTop, autoScroll = false
		_, _ = widget.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyHome}))
		assert.False(t, widget.autoScroll)

		// End key -> GotoBottom, autoScroll = true
		_, _ = widget.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnd}))
		assert.True(t, widget.autoScroll)

		// Up key -> ScrollUp
		_, _ = widget.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyUp}))
		assert.False(t, widget.autoScroll) // Not at bottom anymore

		// Down key -> ScrollDown
		_, _ = widget.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
		// Now it should be at bottom again because it scrolled down 1 line and there were only 5
		// extra lines? Wait, 10 lines total, height 5. GotoBottom = at line 10. ScrollUp = at line 9.
		// ScrollDown = at line 10 (bottom).
		assert.True(t, widget.autoScroll)
	})

	t.Run("View and SetHeight work correctly", func(t *testing.T) {
		widget, err := NewHistoryWidget(keys)
		require.NoError(t, err)

		defer cleanup(widget)

		widget.termWidth = 80
		widget.vp.SetWidth(80)
		widget.SetHeight(10)
		_, _ = widget.Update(outputLineMsg("line 1"))
		_, _ = widget.Update(outputLineMsg("line 2"))

		view := widget.View()
		assert.Contains(t, view.Content, "line 1")
		assert.Contains(t, view.Content, "line 2")
		assert.Contains(t, view.Content, "2 lines") // from status bar
	})
}
