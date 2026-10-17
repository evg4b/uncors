package uncorsapp

import (
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errBoom = errors.New("boom")

type testApp struct {
	*UncorsApp

	bus      *events.Bus
	commands chan events.Command
}

func newTestApp(t *testing.T) testApp {
	t.Helper()

	bus := events.NewBus()
	commands := make(chan events.Command, 1)
	app := NewUncorsApp(bus.Subscribe(10), commands, "x.x.x")

	t.Cleanup(func() {
		bus.Close()
		require.NoError(t, app.historyWidget.Close())
	})

	return testApp{UncorsApp: app, bus: bus, commands: commands}
}

func historyText(app testApp) string {
	return ansi.Strip(strings.Join(app.historyWidget.hist.Window(0, historyCapacity), "\n"))
}

func mappings() config.Mappings {
	return config.Mappings{{From: hosts.Localhost.HTTPPort(3000), To: hosts.Github.HTTPS()}}
}

func TestNewUncorsAppAndKeyMap(t *testing.T) {
	app := newTestApp(t)

	assert.NotNil(t, app.events)
	assert.NotNil(t, app.historyWidget.hist)
	assert.True(t, app.historyWidget.autoScroll)
	assert.Empty(t, app.trackerWidget.pending)
	assert.GreaterOrEqual(t, app.memWidget.memMB, 0.0)
	require.NotNil(t, app.Init())

	keys := NewKeyMap()
	assert.Len(t, keys.ShortHelp(), 3)
	fullHelp := keys.FullHelp()
	require.Len(t, fullHelp, 3)
	assert.Len(t, fullHelp[0], 4)
	assert.Len(t, fullHelp[1], 2)
	assert.Len(t, fullHelp[2], 3)
}

func TestUncorsAppUpdateViewAndLayout(t *testing.T) {
	app := newTestApp(t)

	model, cmd := app.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	require.Same(t, app.UncorsApp, model)
	assert.Nil(t, cmd)
	assert.Equal(t, 80, app.termWidth)
	assert.Equal(t, 12, app.termHeight)

	requestURL, err := url.Parse("https://example.com/demo")
	require.NoError(t, err)

	model, cmd = app.Update(requestStartedMsg{
		ID:      7,
		Method:  "GET",
		URL:     requestURL,
		Handler: events.HandlerProxy,
		At:      time.Now().Add(-1500 * time.Millisecond),
	})
	require.Same(t, app.UncorsApp, model)
	require.NotNil(t, cmd)
	assert.Len(t, app.trackerWidget.pending, 1)
	assert.True(t, app.trackerWidget.ticking)

	view := app.View()
	assert.True(t, view.AltScreen)
	assert.Contains(t, view.Content, "GET")
	assert.Contains(t, view.Content, "example.com/demo")
	assert.Contains(t, view.Content, "PROXY")

	model, cmd = app.Update(requestCompletedMsg{
		ID:      7,
		Handler: events.HandlerProxy,
		Data:    &contracts.RequestData{Method: "GET", URL: requestURL, Code: 200},
	})
	require.Same(t, app.UncorsApp, model)
	require.NotNil(t, cmd)
	assert.Empty(t, app.trackerWidget.pending)
	assert.Contains(t, historyText(app), "PROXY")
	assert.Contains(t, historyText(app), "200 GET")

	model, cmd = app.Update(spinner.TickMsg{})
	require.Same(t, app.UncorsApp, model)
	assert.Nil(t, cmd)
	assert.False(t, app.trackerWidget.ticking)

	model, cmd = app.Update(memUpdateMsg{mb: 12.5})
	require.Same(t, app.UncorsApp, model)
	require.NotNil(t, cmd)
	assert.InDelta(t, 12.5, app.memWidget.memMB, 0.0001)

	app.helpWidget.help.ShowAll = true
	app.trackerWidget.pending[1] = events.RequestStarted{Method: "POST", URL: requestURL, At: time.Now()}
	assert.Equal(t, 6, app.footerHeight())

	app.termHeight = 0
	app.updateHistoryHeight()

	app.termWidth = 120
	assert.Contains(t, app.memWidget.View().Content, "MB")

	app.termWidth = 1
	assert.Equal(t, app.helpWidget.help.View(app.keys), app.helpWidget.View().Content)
}

func TestUncorsAppRendersLifecycleEvents(t *testing.T) {
	t.Run("server started shows the banner and mappings", func(t *testing.T) {
		app := newTestApp(t)

		_, cmd := app.Update(lifecycleMsg{event: events.ServerStarted{Mappings: mappings()}})
		require.NotNil(t, cmd)

		assert.Contains(t, historyText(app), "DON'T USE IT FOR PRODUCTION!")
		assert.Contains(t, historyText(app), "http://localhost:3000 => https://github.com")
	})

	t.Run("server restarted clears pending requests", func(t *testing.T) {
		app := newTestApp(t)
		app.trackerWidget.pending[1] = events.RequestStarted{Method: "GET", At: time.Now()}
		app.trackerWidget.ticking = true

		_, cmd := app.Update(lifecycleMsg{event: events.ServerRestarted{Mappings: mappings()}})
		require.NotNil(t, cmd)

		assert.Contains(t, historyText(app), "Server restarted")
		assert.Empty(t, app.trackerWidget.pending)
		assert.False(t, app.trackerWidget.ticking)
	})

	t.Run("a failed reload is reported", func(t *testing.T) {
		app := newTestApp(t)

		_, _ = app.Update(lifecycleMsg{event: events.ReloadFailed{Err: errBoom}})

		assert.Contains(t, historyText(app), "Failed to reload configuration, keeping the previous one: boom")
	})

	t.Run("a new version is announced", func(t *testing.T) {
		app := newTestApp(t)

		_, _ = app.Update(lifecycleMsg{event: events.NewVersionAvailable{Current: "0.6.0", Latest: "0.7.0"}})

		assert.Contains(t, historyText(app), "0.6.0 is not the latest version, you should upgrade to 0.7.0")
	})

	t.Run("server stopped quits", func(t *testing.T) {
		app := newTestApp(t)

		_, cmd := app.Update(lifecycleMsg{event: events.ServerStopped{}})
		require.NotNil(t, cmd)
		assert.Equal(t, tea.Quit(), cmd())
	})

	t.Run("a closed subscription quits", func(t *testing.T) {
		app := newTestApp(t)

		_, cmd := app.Update(eventsClosedMsg{})
		require.NotNil(t, cmd)
		assert.Equal(t, tea.Quit(), cmd())
	})
}

func TestUncorsAppWatchEvents(t *testing.T) {
	app := newTestApp(t)

	requestURL, err := url.Parse("https://example.com/watch")
	require.NoError(t, err)

	started := events.RequestStarted{ID: 9, Method: "GET", URL: requestURL, Handler: events.HandlerCache}
	completed := events.RequestCompleted{ID: 9, Handler: events.HandlerCache}
	restarted := events.ServerRestarted{Mappings: mappings()}

	app.bus.Publish(started)
	app.bus.Publish(completed)
	app.bus.Publish(restarted)

	assert.Equal(t, requestStartedMsg(started), app.watchEventsCmd()())
	assert.Equal(t, requestCompletedMsg(completed), app.watchEventsCmd()())
	assert.Equal(t, lifecycleMsg{event: restarted}, app.watchEventsCmd()())

	app.bus.Close()
	assert.Equal(t, eventsClosedMsg{}, app.watchEventsCmd()())
}

func TestUncorsAppOverflowedSubscription(t *testing.T) {
	app := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	requestURL, err := url.Parse("https://example.com/overflow")
	require.NoError(t, err)

	for id := range uint64(30) {
		app.bus.Publish(events.RequestStarted{ID: id, Method: "GET", URL: requestURL, Handler: events.HandlerProxy})
	}

	app.bus.Publish(events.ReloadFailed{Err: errBoom})

	for id := range uint64(30) {
		app.bus.Publish(events.RequestCompleted{
			ID:      id,
			Handler: events.HandlerProxy,
			Data:    &contracts.RequestData{Method: "GET", URL: requestURL, Code: 200},
		})
	}

	app.bus.Publish(events.ServerStopped{})
	app.bus.Close()

	for msg := app.watchEventsCmd()(); msg != (lifecycleMsg{event: events.ServerStopped{}}); msg = app.watchEventsCmd()() {
		require.NotEqual(t, eventsClosedMsg{}, msg)
		app.Update(msg)
	}

	assert.Equal(t, eventsClosedMsg{}, app.watchEventsCmd()())
	assert.Empty(t, app.trackerWidget.pending)
	assert.Contains(t, historyText(app), "Failed to reload configuration")
	assert.Equal(t, 10, strings.Count(historyText(app), "200 GET"))
	assert.Equal(t, uint64(40), app.events.Dropped())
}

func TestUncorsAppKeyHandling(t *testing.T) {
	app := newTestApp(t)

	_, _ = app.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	_, _ = app.Update(outputLineMsg("one\ntwo\nthree\nfour\nfive"))

	_, cmd := app.Update(tea.KeyPressMsg(tea.Key{Text: "?", Code: '?'}))
	assert.Nil(t, cmd)
	assert.True(t, app.helpWidget.help.ShowAll)

	_, cmd = app.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	assert.Nil(t, cmd)

	_, cmd = app.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyUp}))
	assert.Nil(t, cmd)

	_, cmd = app.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyPgDown}))
	assert.Nil(t, cmd)

	_, cmd = app.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyPgUp}))
	assert.Nil(t, cmd)

	_, cmd = app.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyHome}))
	assert.Nil(t, cmd)
	assert.False(t, app.historyWidget.autoScroll)

	_, cmd = app.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnd}))
	assert.Nil(t, cmd)
	assert.True(t, app.historyWidget.autoScroll)

	_, cmd = app.Update(tea.KeyPressMsg(tea.Key{Text: "r", Code: 'r'}))
	require.NotNil(t, cmd)
	assert.Nil(t, cmd())
	assert.Equal(t, events.Restart{}, <-app.commands)

	_, cmd = app.Update(tea.KeyPressMsg(tea.Key{Text: "q", Code: 'q'}))
	require.NotNil(t, cmd)
	assert.Nil(t, cmd())
	assert.Equal(t, events.Shutdown{}, <-app.commands)
}

func TestUncorsAppCommandsDoNotBlockAfterStop(t *testing.T) {
	app := newTestApp(t)

	app.commands <- events.Restart{}

	_, cmd := app.Update(tea.KeyPressMsg(tea.Key{Text: "q", Code: 'q'}))
	require.NotNil(t, cmd)

	done := make(chan struct{})

	go func() {
		cmd()
		close(done)
	}()

	_, _ = app.Update(lifecycleMsg{event: events.ServerStopped{}})

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("sending a command blocked after the proxy stopped")
	}
}

func TestEventMsg(t *testing.T) {
	assert.IsType(t, requestStartedMsg{}, EventMsg(events.RequestStarted{ID: 1}))
	assert.IsType(t, requestCompletedMsg{}, EventMsg(events.RequestCompleted{ID: 1}))
	assert.Equal(t, lifecycleMsg{event: events.ServerStopped{}}, EventMsg(events.ServerStopped{}))
}

func TestWithMemoryWidget(t *testing.T) {
	widget := NewMemoryWidget(WithMemoryUsage(func() float64 { return 3 }))

	app := NewUncorsApp(events.NewBus().Subscribe(1), make(chan events.Command, 1), "test", WithMemoryWidget(widget))

	assert.Same(t, widget, app.memWidget)
}
