package uncorsapp

import (
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/spinner"
	"github.com/charmbracelet/x/ansi"
	"github.com/evg4b/uncors/internal/events"
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

	t.Run("Update adds started requests", func(t *testing.T) {
		widget := NewTrackerWidget()

		reqURL, _ := url.Parse("http://localhost/test")
		msg := requestStartedMsg{
			ID:     1,
			Method: "GET",
			URL:    reqURL,
			At:     time.Now(),
		}

		newWidget, cmd := widget.Update(msg)
		assert.Same(t, widget, newWidget)
		require.NotNil(t, cmd)
		assert.True(t, widget.ticking)
		assert.Equal(t, 1, widget.ActiveCount())
		assert.Equal(t, 2, widget.Height())
	})

	t.Run("Update removes completed requests", func(t *testing.T) {
		widget := NewTrackerWidget()
		widget.pending[1] = events.RequestStarted{ID: 1}
		widget.ticking = true

		msg := requestCompletedMsg{ID: 1}
		newWidget, cmd := widget.Update(msg)

		assert.Same(t, widget, newWidget)
		assert.Nil(t, cmd)
		assert.Equal(t, 0, widget.ActiveCount())
	})

	t.Run("Update handles tickMsg when pending requests exist", func(t *testing.T) {
		widget := NewTrackerWidget()
		widget.pending[1] = events.RequestStarted{ID: 1}
		widget.ticking = true

		newWidget, cmd := widget.Update(spinner.TickMsg{})
		assert.Same(t, widget, newWidget)
		assert.NotNil(t, cmd)
		assert.True(t, widget.ticking)
	})

	t.Run("Update handles tickMsg when no pending requests exist", func(t *testing.T) {
		widget := NewTrackerWidget()
		widget.ticking = true

		newWidget, cmd := widget.Update(spinner.TickMsg{})
		assert.Same(t, widget, newWidget)
		assert.Nil(t, cmd)
		assert.False(t, widget.ticking)
	})

	t.Run("Update clears pending requests on restart", func(t *testing.T) {
		widget := NewTrackerWidget()
		widget.pending[1] = events.RequestStarted{ID: 1}
		widget.ticking = true

		newWidget, cmd := widget.Update(lifecycleMsg{event: events.ServerRestarted{}})
		assert.Same(t, widget, newWidget)
		assert.Nil(t, cmd)
		assert.Equal(t, 0, widget.ActiveCount())
		assert.False(t, widget.ticking)
	})

	t.Run("Update keeps pending requests on other lifecycle events", func(t *testing.T) {
		widget := NewTrackerWidget()
		widget.pending[1] = events.RequestStarted{ID: 1}

		_, _ = widget.Update(lifecycleMsg{event: events.ReloadFailed{}})
		assert.Equal(t, 1, widget.ActiveCount())
	})

	t.Run("View renders correctly", func(t *testing.T) {
		widget := NewTrackerWidget()
		reqURL, _ := url.Parse("http://localhost/test")
		widget.pending[1] = events.RequestStarted{
			ID:      1,
			Method:  "POST",
			URL:     reqURL,
			Handler: events.HandlerScript,
			Rewrite: &events.Rewrite{From: "/test", To: "/v2/test"},
			At:      time.Now().Add(-5 * time.Second),
		}

		view := widget.View()
		assert.Contains(t, view.Content, "SCRPT")
		assert.Contains(t, view.Content, "POST")
		assert.Contains(t, view.Content, "localhost/test")
		assert.Contains(t, view.Content, "→ /v2/test")
	})

	t.Run("View keeps long methods on one line", func(t *testing.T) {
		widget := NewTrackerWidget()
		reqURL, _ := url.Parse("http://localhost/test")

		for id, method := range []string{"OPTIONS", "PATCH", "PROPFIND"} {
			widget.pending[uint64(id)] = events.RequestStarted{
				ID: uint64(id), Method: method, URL: reqURL, Handler: events.HandlerOptions,
			}
		}

		assert.Len(t, strings.Split(widget.View().Content, "\n"), 3)
	})

	t.Run("View renders one request per line in ID order", func(t *testing.T) {
		widget := NewTrackerWidget()

		for _, id := range []uint64{3, 1, 2} {
			reqURL, _ := url.Parse("http://localhost/" + strconv.FormatUint(id, 10))
			widget.pending[id] = events.RequestStarted{ID: id, Method: "GET", URL: reqURL}
		}

		lines := strings.Split(ansi.Strip(widget.View().Content), "\n")

		require.Len(t, lines, 3)
		assert.True(t, strings.HasSuffix(lines[0], "localhost/1"), lines[0])
		assert.True(t, strings.HasSuffix(lines[1], "localhost/2"), lines[1])
		assert.True(t, strings.HasSuffix(lines[2], "localhost/3"), lines[2])
	})

	t.Run("View renders empty string when no requests", func(t *testing.T) {
		widget := NewTrackerWidget()
		view := widget.View()
		assert.Empty(t, view.Content)
	})
}
