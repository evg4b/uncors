package uncorsapp

import (
	"errors"
	"net/url"
	"regexp"
	"testing"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/evg4b/uncors/internal/app"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/di"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errBoom = errors.New("boom")

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// stripANSI removes styling so assertions can talk about the text. Lip Gloss
// styles URLs one character at a time, so the plain string is never a
// contiguous substring of the rendered line.
func stripANSI(value string) string {
	return ansiPattern.ReplaceAllString(value, "")
}

// newTestApp builds the model over a real service and an in-memory container,
// and reports how many times the configuration was reloaded.
func newTestApp(t *testing.T) (*UncorsApp, *di.Container, *int) {
	t.Helper()

	cfg := &config.UncorsConfig{Mappings: config.Mappings{}}
	container := di.NewContainer()

	loadCalls := 0
	// No config file path, so the service creates no watcher.
	model := NewUncorsApp(container, "", cfg, func() (*config.UncorsConfig, error) {
		loadCalls++

		return cfg, nil
	})

	t.Cleanup(func() {
		require.NoError(t, model.service.Close())
		require.NoError(t, model.service.Shutdown(t.Context()))
		require.NoError(t, model.historyWidget.Close())
		require.NoError(t, container.Close())
	})

	return model, container, &loadCalls
}

func TestNewUncorsAppAndKeyMap(t *testing.T) {
	app, _, _ := newTestApp(t)

	assert.NotNil(t, app.output)
	assert.NotNil(t, app.renderer)
	assert.NotNil(t, app.historyWidget.hist)
	assert.NotNil(t, app.service)
	assert.NotNil(t, app.done)
	assert.True(t, app.historyWidget.autoScroll)
	assert.Empty(t, app.trackerWidget.pending)
	assert.GreaterOrEqual(t, app.memWidget.memMB, 0.0)
	require.NotNil(t, app.Init())

	keys := newKeyMap()
	assert.Len(t, keys.ShortHelp(), 3)
	fullHelp := keys.FullHelp()
	require.Len(t, fullHelp, 3)
	assert.Len(t, fullHelp[0], 4)
	assert.Len(t, fullHelp[1], 2)
	assert.Len(t, fullHelp[2], 3)
}

func TestUncorsAppUpdateViewAndLayout(t *testing.T) {
	app, _, _ := newTestApp(t)

	model, cmd := app.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	require.Same(t, app, model)
	assert.Nil(t, cmd)
	assert.Equal(t, 80, app.termWidth)
	assert.Equal(t, 12, app.termHeight)

	model, cmd = app.Update(outputLineMsg("hello\nworld"))
	require.Same(t, app, model)
	require.NotNil(t, cmd)
	assert.Equal(t, 2, app.historyWidget.hist.LineCount())
	assert.Equal(t, []string{"hello", "world"}, app.historyWidget.hist.Lines())

	requestURL, err := url.Parse("https://example.com/demo")
	require.NoError(t, err)

	model, cmd = app.Update(requestEventMsg{
		ID:        7,
		Method:    "GET",
		URL:       requestURL,
		StartedAt: time.Now().Add(-1500 * time.Millisecond),
	})
	require.Same(t, app, model)
	require.NotNil(t, cmd) // the spinner starts ticking
	assert.Len(t, app.trackerWidget.pending, 1)
	assert.True(t, app.trackerWidget.ticking)

	view := app.View()
	assert.True(t, view.AltScreen)
	assert.Contains(t, view.Content, "hello")
	assert.Contains(t, view.Content, "world")
	assert.Contains(t, view.Content, "GET")
	assert.Contains(t, view.Content, "example.com/demo")

	model, _ = app.Update(requestEventMsg{ID: 7, Done: true})
	require.Same(t, app, model)
	assert.Empty(t, app.trackerWidget.pending)

	model, cmd = app.Update(spinner.TickMsg{})
	require.Same(t, app, model)
	assert.Nil(t, cmd)
	assert.False(t, app.trackerWidget.ticking)

	model, cmd = app.Update(memUpdateMsg{mb: 12.5})
	require.Same(t, app, model)
	require.NotNil(t, cmd)
	assert.InDelta(t, 12.5, app.memWidget.memMB, 0.0001)

	// The footer grows with the help pane and the in-flight list, and the
	// scrollback is sized from whatever it leaves.
	app.helpWidget.help.ShowAll = true
	app.trackerWidget.pending[1] = server.RequestEvent{Method: "POST", URL: requestURL, StartedAt: time.Now()}
	assert.Equal(t, 6, app.footerHeight())

	app.updateHistoryHeight()
	assert.Equal(t, max(app.termHeight-app.footerHeight(), 1), app.historyHeight)

	// Nothing is sized before the terminal size is known.
	app.termHeight = 0
	app.historyHeight = 0
	app.updateHistoryHeight()
	assert.Zero(t, app.historyHeight)

	assert.Contains(t, app.memWidget.View().Content, "MB")

	// Too narrow for the memory readout, so only the help bar is drawn.
	app.termWidth = 1
	assert.Equal(t, app.helpWidget.help.View(app.keys), app.helpWidget.View().Content)
}

func TestUncorsAppCommandFactoriesAndChannels(t *testing.T) {
	t.Run("start and lifecycle commands return expected messages", func(t *testing.T) {
		app, _, loadCalls := newTestApp(t)

		msg := app.startServerCmd()()
		assert.IsType(t, serverStartedMsg{}, msg)

		assert.Nil(t, app.restartCmd()())
		assert.Equal(t, 1, *loadCalls)

		msg = app.shutdownCmd()()
		assert.Equal(t, shutdownMsg{}, msg)
	})

	t.Run("waitOutputCmd reads from output channel and handles shutdown", func(t *testing.T) {
		app, _, _ := newTestApp(t)

		app.outputCh <- "queued"

		assert.Equal(t, outputLineMsg("queued"), app.waitOutputCmd()())

		require.NoError(t, app.service.Close())
		assert.Nil(t, app.waitOutputCmd()())
	})

	t.Run("waitOutputCmd returns nil when channel is closed", func(t *testing.T) {
		app, _, _ := newTestApp(t)

		close(app.outputCh)
		assert.Nil(t, app.waitOutputCmd()())
	})

	t.Run("request activity arrives through the service stream", func(t *testing.T) {
		model, container, _ := newTestApp(t)

		requestURL, err := url.Parse("https://example.com/watch")
		require.NoError(t, err)

		emitted := server.RequestEvent{ID: 9, Method: "GET", URL: requestURL}
		container.RequestTracker().Emit(emitted)

		// The service is the single consumer of the tracker; the model sees
		// activity only because the service republishes it.
		msg := model.waitServiceEventCmd()()

		event, ok := msg.(serviceEventMsg)
		require.True(t, ok)

		request, ok := event.event.(app.RequestEvent)
		require.True(t, ok)
		assert.Equal(t, emitted, request.Event)

		assert.Equal(t, requestEventMsg(emitted), widgetMessage(request))
	})

	t.Run("waitServiceEventCmd returns nil once the service is closed", func(t *testing.T) {
		model, _, _ := newTestApp(t)

		require.NoError(t, model.service.Close())
		assert.Nil(t, model.waitServiceEventCmd()())
	})
}

func TestUncorsAppKeyHandlingAndMessages(t *testing.T) {
	app, _, _ := newTestApp(t)

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
	// Reload is a command, not a result: the widgets learn about it from the
	// service's StateReloaded event, which is what makes a file-triggered
	// reload behave identically to this key.
	assert.Nil(t, cmd())

	_, cmd = app.Update(tea.KeyPressMsg(tea.Key{Text: "q", Code: 'q'}))
	require.NotNil(t, cmd)
	assert.Equal(t, shutdownMsg{}, cmd())
}

func TestUncorsAppServerErrorRestartShutdownAndFormatting(t *testing.T) {
	t.Run("server error and restart messages update state", func(t *testing.T) {
		app, _, _ := newTestApp(t)

		app.trackerWidget.pending[1] = server.RequestEvent{Method: "GET", StartedAt: time.Now()}
		app.trackerWidget.ticking = true

		model, cmd := app.Update(serverErrMsg{err: errBoom})
		require.Same(t, app, model)
		require.NotNil(t, cmd)
		assert.Contains(t, app.historyWidget.hist.Lines()[0], errBoom.Error())

		model, cmd = app.Update(restartMsg{})
		require.Same(t, app, model)
		assert.Nil(t, cmd)
		assert.Empty(t, app.trackerWidget.pending)
		assert.False(t, app.trackerWidget.ticking)
	})

	// P3: a reload the user did not trigger has to reach the widgets too. The
	// service reports it as a lifecycle event, which is translated here into the
	// same message the restart key produces, so a config file save and the 'r'
	// key clear the in-flight list identically.
	t.Run("a reload reported by the service clears the in-flight list", func(t *testing.T) {
		model, _, _ := newTestApp(t)

		model.trackerWidget.pending[1] = server.RequestEvent{Method: "GET", StartedAt: time.Now()}
		model.trackerWidget.ticking = true

		_, _ = model.Update(serviceEventMsg{
			event: app.LifecycleEvent{State: app.StateReloaded},
		})

		assert.Empty(t, model.trackerWidget.pending)
		assert.False(t, model.trackerWidget.ticking)
	})

	t.Run("shutdown message quits app", func(t *testing.T) {
		app, _, _ := newTestApp(t)
		app.historyWidget.hist.AppendLine("hello")

		model, cmd := app.Update(shutdownMsg{})
		require.Same(t, app, model)
		require.NotNil(t, cmd)
		assert.Equal(t, tea.Quit(), cmd())
	})
}

func TestRequestEventsAreRenderedIntoHistory(t *testing.T) {
	requestURL, err := url.Parse("https://example.com/api")
	require.NoError(t, err)

	data := &contracts.RequestData{Method: "GET", URL: requestURL, Code: 200}

	testCases := []struct {
		name   string
		prefix string
	}{
		{name: "without prefix"},
		{name: "with prefix", prefix: "api"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			model, _, _ := newTestApp(t)

			// Rendering lives in internal/render now; the model just hands the
			// event over and the line lands on the output channel.
			model.renderer.Render(app.RequestEvent{
				Event: server.RequestEvent{Done: true, Data: data, Prefix: testCase.prefix},
			})

			require.NotEmpty(t, model.outputCh)
			assert.Contains(t, stripANSI(<-model.outputCh), "example.com/api")
		})
	}
}

// A config that fails to parse or validate must leave the running generation
// serving, exactly as headless mode does. Before this was fixed the failing
// load produced a nil config, which BuildRuntime dereferenced.
func TestReloadWithFailingConfigLoadKeepsServing(t *testing.T) {
	port := testutils.GetFreePort(t)
	cfg := &config.UncorsConfig{
		Mappings: config.Mappings{
			{From: hosts.Localhost.HTTPPort(port), To: hosts.Localhost.HTTP()},
		},
	}

	container := di.NewContainer()

	loadCalls := 0
	model := NewUncorsApp(container, "", cfg, func() (*config.UncorsConfig, error) {
		loadCalls++

		return nil, errBoom
	})

	t.Cleanup(func() {
		require.NoError(t, model.service.Close())
		require.NoError(t, model.service.Shutdown(t.Context()))
		require.NoError(t, container.Close())
	})

	require.NoError(t, model.service.Start(model.service.Context()))

	require.NotPanics(t, func() {
		assert.Nil(t, model.restartCmd()())
	})

	assert.Equal(t, 1, loadCalls)
	assert.False(t, testutils.IsPortFree(port), "the previous generation must still be bound")

	// The model's own interface is deliberately narrow, so reach for the
	// concrete service to assert on the state it recorded.
	service, ok := model.service.(*app.Service)
	require.True(t, ok)

	status := service.Status()
	assert.Equal(t, app.StateReloadFailed, status.State)
	require.ErrorIs(t, status.Err, errBoom)
}
