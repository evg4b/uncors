package tuiapp

import (
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHistoryWidget(t *testing.T) {
	keys := NewKeyMap()

	cleanup := func(widget *HistoryWidget) {
		if widget != nil && widget.hist != nil {
			err := widget.Close()
			require.NoError(t, err)
		}
	}

	t.Run("NewHistoryWidget initializes correctly", func(t *testing.T) {
		widget := NewHistoryWidget(keys)

		defer cleanup(widget)

		assert.NotNil(t, widget)
		assert.NotNil(t, widget.hist)
		assert.True(t, widget.autoScroll)
		assert.False(t, widget.HasLines())
		assert.Nil(t, widget.Init())
	})

	t.Run("Update handles tea.WindowSizeMsg", func(t *testing.T) {
		widget := NewHistoryWidget(keys)

		defer cleanup(widget)

		msg := tea.WindowSizeMsg{Width: 100, Height: 50}
		newWidget, cmd := widget.Update(msg)

		assert.Same(t, widget, newWidget)
		assert.Nil(t, cmd)
		assert.Equal(t, 100, widget.termWidth)
	})

	t.Run("AppendLine adds the lines to the view", func(t *testing.T) {
		widget := NewHistoryWidget(keys)
		widget.Update(tea.WindowSizeMsg{Width: 40, Height: 5})
		widget.SetHeight(5)

		widget.AppendLine("first\nsecond")

		assert.True(t, widget.HasLines())
		assert.Contains(t, widget.View().Content, "second")
	})

	t.Run("Update handles outputLineMsg", func(t *testing.T) {
		widget := NewHistoryWidget(keys)

		defer cleanup(widget)

		msg := outputLineMsg("hello world")
		newWidget, cmd := widget.Update(msg)

		assert.Same(t, widget, newWidget)
		assert.Nil(t, cmd)
		assert.True(t, widget.HasLines())
		assert.Equal(t, 1, widget.hist.LineCount())
	})

	t.Run("Update handles key presses", func(t *testing.T) {
		widget := NewHistoryWidget(keys)

		defer cleanup(widget)

		widget.termWidth = 80
		widget.Update(tea.WindowSizeMsg{Width: 80})
		widget.SetHeight(5)

		for range 10 {
			_, _ = widget.Update(outputLineMsg("line"))
		}

		widget.autoScroll = true

		_, _ = widget.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyHome}))
		assert.False(t, widget.autoScroll)

		_, _ = widget.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnd}))
		assert.True(t, widget.autoScroll)

		_, _ = widget.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyUp}))
		assert.False(t, widget.autoScroll)

		_, _ = widget.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
		assert.True(t, widget.autoScroll)
	})

	t.Run("View and SetHeight work correctly", func(t *testing.T) {
		widget := NewHistoryWidget(keys)

		defer cleanup(widget)

		widget.termWidth = 80
		widget.Update(tea.WindowSizeMsg{Width: 80})
		widget.SetHeight(10)
		_, _ = widget.Update(outputLineMsg("line 1"))
		_, _ = widget.Update(outputLineMsg("line 2"))

		view := widget.View()
		assert.Contains(t, view.Content, "line 1")
		assert.Contains(t, view.Content, "line 2")
	})

	t.Run("keeps the scrolled position while history is evicted", func(t *testing.T) {
		widget := NewHistoryWidget(keys)

		defer cleanup(widget)

		widget.Update(tea.WindowSizeMsg{Width: 40, Height: 3})
		widget.SetHeight(3)

		for index := range historyCapacity {
			widget.AppendLine(strconv.Itoa(index))
		}

		_, _ = widget.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyUp}))
		assert.False(t, widget.autoScroll)
		assert.Equal(t, []string{"9996", "9997", "9998"}, strings.Fields(widget.View().Content))

		widget.AppendLine("next")
		widget.AppendLine("last")

		assert.Equal(t, historyCapacity, widget.hist.LineCount())
		assert.Equal(t, []string{"9996", "9997", "9998"}, strings.Fields(widget.View().Content))

		_, _ = widget.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnd}))
		assert.Equal(t, []string{"9999", "next", "last"}, strings.Fields(widget.View().Content))
	})

	t.Run("cuts lines wider than the terminal", func(t *testing.T) {
		widget := NewHistoryWidget(keys)

		defer cleanup(widget)

		widget.Update(tea.WindowSizeMsg{Width: 5, Height: 1})
		widget.SetHeight(1)
		widget.AppendLine("0123456789")

		assert.Equal(t, "01234", widget.View().Content)
	})
}
