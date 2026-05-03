package uncorsapp

import (
	"net/url"
	"testing"
	"time"

	"charm.land/bubbles/v2/spinner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrackerWidget(t *testing.T) {
	t.Run("NewTrackerWidget initializes correctly", func(t *testing.T) {
		widget := NewTrackerWidget()
		assert.NotNil(t, widget)
		assert.NotNil(t, widget.pending)
		assert.False(t, widget.ticking)
		assert.Equal(t, 0, widget.ActiveCount())
		assert.Equal(t, 0, widget.Height())
		assert.Nil(t, widget.Init())
	})

	t.Run("Update handles requestEventMsg additions", func(t *testing.T) {
		widget := NewTrackerWidget()

		reqURL, _ := url.Parse("http://localhost/test")
		msg := requestEventMsg{
			id:        1,
			method:    "GET",
			url:       reqURL,
			startedAt: time.Now(),
			done:      false,
		}

		newWidget, cmd := widget.Update(msg)
		assert.Same(t, widget, newWidget)
		require.NotNil(t, cmd) // tick command should be returned
		assert.True(t, widget.ticking)
		assert.Equal(t, 1, widget.ActiveCount())
		assert.Equal(t, 2, widget.Height()) // header + 1 request
	})

	t.Run("Update handles requestEventMsg removals", func(t *testing.T) {
		widget := NewTrackerWidget()
		widget.pending[1] = requestEvent{id: 1}
		widget.ticking = true

		msg := requestEventMsg{id: 1, done: true}
		newWidget, cmd := widget.Update(msg)

		assert.Same(t, widget, newWidget)
		assert.Nil(t, cmd) // No new tick command for removal
		assert.Equal(t, 0, widget.ActiveCount())
	})

	t.Run("Update handles tickMsg when pending requests exist", func(t *testing.T) {
		widget := NewTrackerWidget()
		widget.pending[1] = requestEvent{id: 1}
		widget.ticking = true

		newWidget, cmd := widget.Update(spinner.TickMsg{})
		assert.Same(t, widget, newWidget)
		assert.NotNil(t, cmd) // Continues ticking
		assert.True(t, widget.ticking)
	})

	t.Run("Update handles tickMsg when no pending requests exist", func(t *testing.T) {
		widget := NewTrackerWidget()
		widget.ticking = true // E.g., just removed last item

		newWidget, cmd := widget.Update(spinner.TickMsg{})
		assert.Same(t, widget, newWidget)
		assert.Nil(t, cmd) // Stops ticking
		assert.False(t, widget.ticking)
	})

	t.Run("Update handles restartMsg", func(t *testing.T) {
		widget := NewTrackerWidget()
		widget.pending[1] = requestEvent{id: 1}
		widget.ticking = true

		newWidget, cmd := widget.Update(restartMsg{})
		assert.Same(t, widget, newWidget)
		assert.Nil(t, cmd)
		assert.Equal(t, 0, widget.ActiveCount())
		assert.False(t, widget.ticking)
	})

	t.Run("View renders correctly", func(t *testing.T) {
		widget := NewTrackerWidget()
		reqURL, _ := url.Parse("http://localhost/test")
		widget.pending[1] = requestEvent{
			id:        1,
			method:    "POST",
			url:       reqURL,
			startedAt: time.Now().Add(-5 * time.Second),
		}

		view := widget.View()
		assert.Contains(t, view.Content, "POST")
		assert.Contains(t, view.Content, "localhost/test")
	})

	t.Run("View renders empty string when no requests", func(t *testing.T) {
		widget := NewTrackerWidget()
		view := widget.View()
		assert.Empty(t, view.Content)
	})
}
