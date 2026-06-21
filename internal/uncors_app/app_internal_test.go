package uncorsapp

import (
	"errors"
	"net/url"
	"os"
	"testing"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/di"
	"github.com/evg4b/uncors/internal/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errBoom = errors.New("boom")

func newTestApp(t *testing.T) (*UncorsApp, *int) {
	t.Helper()

	uncorsConfig := &config.UncorsConfig{
		Mappings: config.Mappings{},
	}

	container := di.NewContainer()

	loadCalls := 0
	app := NewUncorsApp(
		"test-version",
		container,
		"", // no config file — watcher is not created
		uncorsConfig,
		func() *config.UncorsConfig {
			loadCalls++

			return uncorsConfig
		},
	)

	return app, &loadCalls
}

func cleanupTestApp(t *testing.T, app *UncorsApp) {
	t.Helper()

	app.cancel()
	err := app.app.Close()
	require.NoError(t, err)

	if app.historyWidget != nil && app.historyWidget.hist != nil {
		err := app.historyWidget.hist.Close()
		require.NoError(t, err)
	}
}

func TestNewUncorsAppAndKeyMap(t *testing.T) {
	app, _ := newTestApp(t)
	defer cleanupTestApp(t, app)

	assert.Equal(t, "test-version", app.version)
	assert.NotNil(t, app.output)
	assert.NotNil(t, app.tracker)
	assert.NotNil(t, app.historyWidget.hist)
	assert.NotNil(t, app.appContext)
	assert.NotNil(t, app.appDone)
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
	app, _ := newTestApp(t)
	defer cleanupTestApp(t, app)

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
	require.NotNil(t, cmd)
	assert.Len(t, app.trackerWidget.pending, 1)
	assert.True(t, app.trackerWidget.ticking)

	view := app.View()
	assert.True(t, view.AltScreen)
	assert.Contains(t, view.Content, "hello")
	assert.Contains(t, view.Content, "world")
	assert.Contains(t, view.Content, "GET")
	assert.Contains(t, view.Content, "example.com/demo")

	model, cmd = app.Update(requestEventMsg{ID: 7, Done: true})
	require.Same(t, app, model)
	require.NotNil(t, cmd)
	assert.Empty(t, app.trackerWidget.pending)

	model, cmd = app.Update(spinner.TickMsg{})
	require.Same(t, app, model)
	assert.Nil(t, cmd)
	assert.False(t, app.trackerWidget.ticking)

	model, cmd = app.Update(memUpdateMsg{mb: 12.5})
	require.Same(t, app, model)
	require.NotNil(t, cmd)
	assert.InDelta(t, 12.5, app.memWidget.memMB, 0.0001)

	app.helpWidget.help.ShowAll = true
	app.trackerWidget.pending[1] = server.RequestEvent{Method: "POST", URL: requestURL, StartedAt: time.Now()}
	assert.Equal(t, 6, app.footerHeight())
	// historyHeight is now calculated dynamically and applied to historyWidget in Update/handleRestart.
	// Since we mock manual property setting here, let's call updateHistoryHeight
	app.updateHistoryHeight()

	app.termHeight = 0
	app.updateHistoryHeight()

	app.termWidth = 120
	// renderHelpBar is gone, HelpWidget and MemWidget composite in View()
	// Let's assert MemWidget produces MB string
	assert.Contains(t, app.memWidget.View().Content, "MB")

	app.termWidth = 1
	assert.Equal(t, app.helpWidget.help.View(app.keys), app.helpWidget.View().Content)
}

func TestUncorsAppCommandFactoriesAndChannels(t *testing.T) {
	t.Run("start and lifecycle commands return expected messages", func(t *testing.T) {
		app, loadCalls := newTestApp(t)
		defer cleanupTestApp(t, app)

		msg := app.startServerCmd()()
		assert.IsType(t, serverStartedMsg{}, msg)

		cmd := app.handleServerStarted()
		require.NotNil(t, cmd)

		msg = app.restartCmd()()
		assert.Equal(t, restartMsg{}, msg)
		assert.Equal(t, 1, *loadCalls)

		msg = app.shutdownCmd()()
		assert.Equal(t, shutdownMsg{}, msg)
	})

	t.Run("waitOutputCmd reads from output channel and handles shutdown", func(t *testing.T) {
		app, _ := newTestApp(t)
		defer cleanupTestApp(t, app)

		app.outputCh <- "queued"

		assert.Equal(t, outputLineMsg("queued"), app.waitOutputCmd()())

		app.cancel()
		assert.Nil(t, app.waitOutputCmd()())
	})

	t.Run("waitOutputCmd returns nil when channel is closed", func(t *testing.T) {
		app, _ := newTestApp(t)
		defer cleanupTestApp(t, app)

		close(app.outputCh)
		assert.Nil(t, app.waitOutputCmd()())
	})

	t.Run("watchEventsCmd reads from event channel and handles shutdown", func(t *testing.T) {
		app, _ := newTestApp(t)
		defer cleanupTestApp(t, app)

		requestURL, err := url.Parse("https://example.com/watch")
		require.NoError(t, err)

		app.tracker.Emit(server.RequestEvent{ID: 9, Method: "GET", URL: requestURL})

		assert.Equal(
			t,
			requestEventMsg(server.RequestEvent{ID: 9, Method: "GET", URL: requestURL}),
			app.watchEventsCmd()(),
		)

		app.cancel()
		assert.Nil(t, app.watchEventsCmd()())
	})

	t.Run("watchEventsCmd returns nil when event channel is closed", func(t *testing.T) {
		app, _ := newTestApp(t)
		defer cleanupTestApp(t, app)

		app.tracker.Close()
		assert.Nil(t, app.watchEventsCmd()())
	})
}

func TestUncorsAppKeyHandlingAndMessages(t *testing.T) {
	app, _ := newTestApp(t)
	defer cleanupTestApp(t, app)

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
	assert.Equal(t, restartMsg{}, cmd())

	_, cmd = app.Update(tea.KeyPressMsg(tea.Key{Text: "q", Code: 'q'}))
	require.NotNil(t, cmd)
	assert.Equal(t, shutdownMsg{}, cmd())
}

func TestUncorsAppServerErrorRestartShutdownAndFormatting(t *testing.T) {
	t.Run("server error and restart messages update state", func(t *testing.T) {
		app, _ := newTestApp(t)
		defer cleanupTestApp(t, app)

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

	t.Run("shutdown message quits app", func(t *testing.T) {
		app, _ := newTestApp(t)
		app.historyWidget.hist.AppendLine("hello")

		model, cmd := app.Update(shutdownMsg{})
		require.Same(t, app, model)
		require.NotNil(t, cmd)
		assert.Equal(t, tea.Quit(), cmd())

		app.cancel()
		err := app.app.Close()
		require.NoError(t, err)
	})
}

func TestServerStartedMsgUpdate(t *testing.T) {
	app, _ := newTestApp(t)
	defer cleanupTestApp(t, app)

	model, cmd := app.Update(serverStartedMsg{})

	require.Same(t, app, model)
	require.NotNil(t, cmd)
}

func TestHandleServerStartedWithConfigPath(t *testing.T) {
	t.Run("creates watcher when config file exists", func(t *testing.T) {
		tmpFile, err := os.CreateTemp(t.TempDir(), "uncors-*.yaml")
		require.NoError(t, err)

		err = tmpFile.Close()
		require.NoError(t, err)

		cfg := &config.UncorsConfig{Mappings: config.Mappings{}}

		container := di.NewContainer()
		app := NewUncorsApp("v1", container, tmpFile.Name(), cfg, func() *config.UncorsConfig { return cfg })

		defer func() {
			app.cancel()
			err := app.app.Close()
			require.NoError(t, err)

			if app.historyWidget != nil && app.historyWidget.hist != nil {
				err := app.historyWidget.hist.Close()
				require.NoError(t, err)
			}
		}()

		cmd := app.handleServerStarted()

		require.NotNil(t, cmd)
		require.NotNil(t, app.watcher)

		err = app.watcher.Close()
		require.NoError(t, err)
	})

	t.Run("logs error when config file does not exist", func(t *testing.T) {
		cfg := &config.UncorsConfig{Mappings: config.Mappings{}}

		container := di.NewContainer()
		app := NewUncorsApp("v1", container, "/nonexistent/path/config.yaml", cfg, func() *config.UncorsConfig { return cfg })

		defer func() {
			app.cancel()
			err := app.app.Close()
			require.NoError(t, err)

			if app.historyWidget != nil && app.historyWidget.hist != nil {
				err := app.historyWidget.hist.Close()
				require.NoError(t, err)
			}
		}()

		cmd := app.handleServerStarted()

		require.NotNil(t, cmd)
		assert.Nil(t, app.watcher)
	})
}

func TestHandleRequestEventWithData(t *testing.T) {
	requestURL, err := url.Parse("https://example.com/api")
	require.NoError(t, err)

	data := &contracts.RequestData{Method: "GET", URL: requestURL, Code: 200}

	t.Run("outputs request without prefix", func(t *testing.T) {
		app, _ := newTestApp(t)
		defer cleanupTestApp(t, app)

		app.handleRequestEvent(requestEventMsg{Done: true, Data: data})
	})

	t.Run("outputs request with prefix", func(t *testing.T) {
		app, _ := newTestApp(t)
		defer cleanupTestApp(t, app)

		app.handleRequestEvent(requestEventMsg{Done: true, Data: data, Prefix: "api"})
	})
}

func TestHandleServerStartedCallbackOnFileChange(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "uncors-*.yaml")
	require.NoError(t, err)

	err = tmpFile.Close()
	require.NoError(t, err)

	cfg := &config.UncorsConfig{Mappings: config.Mappings{}}

	called := make(chan struct{}, 1)

	contaienr := di.NewContainer()
	app := NewUncorsApp("v1", contaienr, tmpFile.Name(), cfg, func() *config.UncorsConfig {
		select {
		case called <- struct{}{}:
		default:
		}

		return cfg
	})

	defer func() {
		// Cancel context first so any in-flight Restart fails fast.
		// We deliberately skip app.app.Close() here: closeAll() writes
		// app.closers concurrently with the Restart goroutine's read of
		// app.closers, which would be a data race.
		app.cancel()

		if app.watcher != nil {
			err := app.watcher.Close()
			require.NoError(t, err)
		}

		if app.historyWidget != nil && app.historyWidget.hist != nil {
			err := app.historyWidget.hist.Close()
			require.NoError(t, err)
		}
	}()

	cmd := app.handleServerStarted()

	require.NotNil(t, cmd)
	require.NotNil(t, app.watcher)

	require.NoError(t, os.WriteFile(tmpFile.Name(), []byte("proxy: \"\""), 0o600))

	select {
	case <-called:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("onChange callback was not invoked within timeout")
	}
}

func TestHandleShutdownWithWatcher(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "uncors-*.yaml")
	require.NoError(t, err)

	err = tmpFile.Close()
	require.NoError(t, err)

	ctx := t.Context()

	watcher := config.NewWatcher(tmpFile.Name())
	err = watcher.Watch(ctx, func() {})
	require.NoError(t, err)

	app, _ := newTestApp(t)
	app.watcher = watcher

	cmd := app.handleShutdown()
	require.NotNil(t, cmd)
	assert.Equal(t, tea.Quit(), cmd())

	app.cancel()
	err = app.app.Close()
	require.NoError(t, err)

	if app.historyWidget != nil && app.historyWidget.hist != nil {
		err := app.historyWidget.hist.Close()
		require.NoError(t, err)
	}
}
