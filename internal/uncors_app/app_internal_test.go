package uncorsapp

import (
	"errors"
	"net/url"
	"os"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/evg4b/uncors/internal/config"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errBoom = errors.New("boom")

func newTestApp(t *testing.T) (*uncorsApp, *int) {
	t.Helper()

	fs := afero.NewMemMapFs()
	viperInstance := viper.New()
	uncorsConfig := &config.UncorsConfig{
		Mappings: config.Mappings{},
	}

	loadCalls := 0
	model := NewUncorsApp(
		"test-version",
		fs,
		viperInstance,
		uncorsConfig,
		func() *config.UncorsConfig {
			loadCalls++

			return uncorsConfig
		},
	)

	app, ok := model.(*uncorsApp)
	require.True(t, ok)

	return app, &loadCalls
}

func cleanupTestApp(t *testing.T, app *uncorsApp) {
	t.Helper()

	app.cancel()
	_ = app.app.Close()

	if app.hist == nil || app.hist.file == nil {
		return
	}

	_, err := os.Stat(app.hist.file.Name())
	if err == nil {
		_ = app.hist.Close()
	}
}

func TestNewUncorsAppAndKeyMap(t *testing.T) {
	app, _ := newTestApp(t)
	defer cleanupTestApp(t, app)

	assert.Equal(t, "test-version", app.version)
	assert.NotNil(t, app.output)
	assert.NotNil(t, app.tracker)
	assert.NotNil(t, app.hist)
	assert.NotNil(t, app.appContext)
	assert.NotNil(t, app.appDone)
	assert.True(t, app.autoScroll)
	assert.Empty(t, app.pending)
	assert.GreaterOrEqual(t, app.memMB, 0.0)
	require.NotNil(t, app.Init())

	keys := newKeyMap()
	assert.Len(t, keys.ShortHelp(), 6)
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
	assert.Equal(t, 2, app.hist.LineCount())
	assert.Equal(t, []string{"hello", "world"}, app.hist.Lines())

	requestURL, err := url.Parse("https://example.com/demo")
	require.NoError(t, err)

	model, cmd = app.Update(requestEventMsg{
		id:        7,
		method:    "GET",
		url:       requestURL,
		startedAt: time.Now().Add(-1500 * time.Millisecond),
	})
	require.Same(t, app, model)
	require.NotNil(t, cmd)
	assert.Len(t, app.pending, 1)
	assert.True(t, app.ticking)

	view := app.View()
	assert.True(t, view.AltScreen)
	assert.Contains(t, view.Content, "hello")
	assert.Contains(t, view.Content, "world")
	assert.Contains(t, view.Content, "In progress (1):")
	assert.Contains(t, view.Content, "GET")
	assert.Contains(t, view.Content, "example.com/demo")
	assert.Contains(t, view.Content, "lines")

	model, cmd = app.Update(requestEventMsg{id: 7, done: true})
	require.Same(t, app, model)
	require.NotNil(t, cmd)
	assert.Empty(t, app.pending)

	model, cmd = app.Update(tickMsg{})
	require.Same(t, app, model)
	assert.Nil(t, cmd)
	assert.False(t, app.ticking)

	model, cmd = app.Update(memUpdateMsg{mb: 12.5})
	require.Same(t, app, model)
	require.NotNil(t, cmd)
	assert.InDelta(t, 12.5, app.memMB, 0.0001)

	app.help.ShowAll = true
	app.pending[1] = requestEvent{method: "POST", url: requestURL, startedAt: time.Now()}
	assert.Equal(t, 6, app.footerHeight())
	assert.Equal(t, 6, app.historyHeight())

	app.termHeight = 0
	assert.Equal(t, 1, app.historyHeight())

	app.termWidth = 120
	assert.Contains(t, app.renderHelpBar(), "MB")

	app.termWidth = 1
	assert.Equal(t, app.help.View(app.keys), app.renderHelpBar())

	app.termWidth = 80
	app.autoScroll = false
	assert.Contains(t, app.renderStatusBar(), "0%")
	assert.NotContains(t, app.renderStatusBar(), "[auto]")
}

func TestUncorsAppCommandFactoriesAndChannels(t *testing.T) {
	t.Run("start and lifecycle commands return expected messages", func(t *testing.T) {
		app, loadCalls := newTestApp(t)
		defer cleanupTestApp(t, app)

		msg := app.startServerCmd()()
		assert.IsType(t, serverStartedMsg{}, msg)

		cmd := app.handleServerStarted()
		require.NotNil(t, cmd)
		app.viper.OnConfigChange(func(_ fsnotify.Event) {})

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

		app.tracker.events <- requestEvent{id: 9, method: "GET", url: requestURL}

		assert.Equal(
			t,
			requestEventMsg(requestEvent{id: 9, method: "GET", url: requestURL}),
			app.watchEventsCmd()(),
		)

		app.cancel()
		assert.Nil(t, app.watchEventsCmd()())
	})

	t.Run("watchEventsCmd returns nil when event channel is closed", func(t *testing.T) {
		app, _ := newTestApp(t)
		defer cleanupTestApp(t, app)

		close(app.tracker.events)
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
	assert.True(t, app.help.ShowAll)

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
	assert.False(t, app.autoScroll)

	_, cmd = app.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnd}))
	assert.Nil(t, cmd)
	assert.True(t, app.autoScroll)

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

		app.pending[1] = requestEvent{method: "GET", startedAt: time.Now()}
		app.ticking = true

		model, cmd := app.Update(serverErrMsg{err: errBoom})
		require.Same(t, app, model)
		require.NotNil(t, cmd)
		assert.Contains(t, app.hist.Lines()[0], errBoom.Error())

		model, cmd = app.Update(restartMsg{})
		require.Same(t, app, model)
		assert.Nil(t, cmd)
		assert.Empty(t, app.pending)
		assert.False(t, app.ticking)
	})

	t.Run("shutdown message closes history file", func(t *testing.T) {
		app, _ := newTestApp(t)
		app.hist.AppendLine("hello")

		historyPath := app.hist.file.Name()
		_, statErr := os.Stat(historyPath)
		require.NoError(t, statErr)

		model, cmd := app.Update(shutdownMsg{})
		require.Same(t, app, model)
		require.NotNil(t, cmd)

		_, statErr = os.Stat(historyPath)
		assert.True(t, os.IsNotExist(statErr))

		app.cancel()
		_ = app.app.Close()
	})

	t.Run("formatElapsed renders milliseconds and seconds", func(t *testing.T) {
		assert.Equal(t, "999ms", formatElapsed(999*time.Millisecond))
		assert.Equal(t, "1.5s", formatElapsed(1500*time.Millisecond))
	})
}
