package uncorsapp

import (
	"context"
	"strings"
	"time"

	key "charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/uncors"
	"github.com/evg4b/uncors/internal/version"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

const (
	outputChannelSize = 1000
	shutdownTimeout   = 5 * time.Second
	versionCheckDelay = 50 * time.Second
	tickInterval      = 200 * time.Millisecond
	memTickInterval   = 2 * time.Second
	bytesPerMegabyte  = 1024 * 1024
)

type uncorsApp struct {
	version string
	keys    keyMap

	app     *uncors.Uncors
	output  *tuiOutput
	tracker *requestTracker

	outputCh   chan string
	appContext func() context.Context
	appDone    <-chan struct{}
	cancel     context.CancelFunc

	cfg        *config.UncorsConfig
	loadConfig func() *config.UncorsConfig
	viper      *viper.Viper

	termHeight int
	termWidth  int

	historyWidget *HistoryWidget
	trackerWidget *TrackerWidget
	helpWidget    *HelpWidget
	memWidget     *MemoryWidget
}

type (
	serverStartedMsg struct{}
	serverErrMsg     struct{ err error }
	shutdownMsg      struct{}
	restartMsg       struct{}
)

type appUpdateMsg interface {
	update(app *uncorsApp) tea.Cmd
}

func NewUncorsApp(
	ver string,
	fs afero.Fs,
	viperInstance *viper.Viper,
	cfg *config.UncorsConfig,
	loadConfig func() *config.UncorsConfig,
) tea.Model {
	outputCh := make(chan string, outputChannelSize)
	output := newTuiOutput(outputCh)
	tracker := newRequestTracker()
	appCtx, cancel := context.WithCancel(context.Background())

	keys := newKeyMap()

	historyWidget := NewHistoryWidget(keys)

	return &uncorsApp{
		version:       ver,
		keys:          keys,
		app:           uncors.CreateUncors(fs, output, ver).WithHandlerWrapper(tracker.Wrap),
		output:        output,
		tracker:       tracker,
		outputCh:      outputCh,
		appContext:    func() context.Context { return appCtx },
		appDone:       appCtx.Done(),
		cancel:        cancel,
		cfg:           cfg,
		loadConfig:    loadConfig,
		viper:         viperInstance,
		historyWidget: historyWidget,
		trackerWidget: NewTrackerWidget(),
		helpWidget:    NewHelpWidget(keys),
		memWidget:     NewMemoryWidget(),
	}
}

func (m *uncorsApp) Init() tea.Cmd {
	return tea.Batch(
		m.startServerCmd(),
		m.waitOutputCmd(),
		m.watchEventsCmd(),
		m.memWidget.Init(),
		m.trackerWidget.Init(),
		m.historyWidget.Init(),
		m.helpWidget.Init(),
	)
}

func (m *uncorsApp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch typedMsg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termHeight = typedMsg.Height
		m.termWidth = typedMsg.Width
		m.updateHistoryHeight()

	case restartMsg:
		m.handleRestart()

	case outputLineMsg:
		cmds = append(cmds, m.waitOutputCmd())

	case requestEventMsg:
		cmds = append(cmds, m.watchEventsCmd())

	case tea.KeyPressMsg:
		if cmd := m.handleKeyPress(typedMsg); cmd != nil {
			return m, cmd
		}
	}

	if appMsg, ok := msg.(appUpdateMsg); ok {
		cmds = append(cmds, appMsg.update(m))
	}

	// Update widgets
	hw, hwCmd := m.historyWidget.Update(msg)
	m.historyWidget = hw

	cmds = append(cmds, hwCmd)

	tw, twCmd := m.trackerWidget.Update(msg)
	m.trackerWidget = tw

	cmds = append(cmds, twCmd)

	hpw, hpwCmd := m.helpWidget.Update(msg)
	m.helpWidget = hpw

	cmds = append(cmds, hpwCmd)

	mw, mwCmd := m.memWidget.Update(msg)
	m.memWidget = mw

	cmds = append(cmds, mwCmd)

	// Re-calculate history height if tracker or help dimensions changed
	m.updateLayout(msg)

	return m, tea.Batch(cmds...)
}

func (m *uncorsApp) View() tea.View {
	var viewBuilder strings.Builder

	// 1. History
	viewBuilder.WriteString(m.historyWidget.View().Content)

	// 2. Tracker (In progress requests)
	if m.trackerWidget.ActiveCount() > 0 {
		viewBuilder.WriteByte('\n')
		viewBuilder.WriteString(m.trackerWidget.View().Content)
	}

	// 3. Help Bar and Memory
	viewBuilder.WriteByte('\n')

	helpStr := m.helpWidget.View().Content
	memStr := m.memWidget.View().Content

	gap := m.termWidth - lipgloss.Width(helpStr) - lipgloss.Width(memStr)
	if gap > 0 {
		viewBuilder.WriteString(helpStr + strings.Repeat(" ", gap) + memStr)
	} else {
		viewBuilder.WriteString(helpStr)
	}

	v := tea.NewView(viewBuilder.String())
	v.AltScreen = true

	return v
}

func (m *uncorsApp) updateLayout(msg tea.Msg) {
	if _, isRequest := msg.(requestEventMsg); isRequest {
		m.updateHistoryHeight()
	} else if _, isKey := msg.(tea.KeyPressMsg); isKey {
		m.updateHistoryHeight()
	}
}

func (m *uncorsApp) handleKeyPress(msg tea.KeyPressMsg) tea.Cmd {
	if key.Matches(msg, m.keys.Restart) {
		return m.restartCmd()
	}

	if key.Matches(msg, m.keys.Quit) {
		return m.shutdownCmd()
	}

	return nil
}

func (m *uncorsApp) updateHistoryHeight() {
	viewportHeight := max(m.termHeight-m.footerHeight(), 1)

	m.historyWidget.SetHeight(viewportHeight)
}

func (m *uncorsApp) footerHeight() int {
	footerHeight := m.helpWidget.Height()

	if m.trackerWidget.ActiveCount() > 0 {
		footerHeight += m.trackerWidget.Height()
	}

	return footerHeight
}

func (msg serverStartedMsg) update(app *uncorsApp) tea.Cmd {
	return app.handleServerStarted()
}

func (msg serverErrMsg) update(app *uncorsApp) tea.Cmd {
	return app.handleServerError(msg)
}

func (msg shutdownMsg) update(app *uncorsApp) tea.Cmd {
	return app.handleShutdown()
}

func (m *uncorsApp) handleServerStarted() tea.Cmd {
	m.viper.OnConfigChange(func(_ fsnotify.Event) {
		defer helpers.PanicInterceptor(func(value any) {
			m.output.Errorf("Config reloading error: %v", value)
		})

		newCfg := m.loadConfig()

		err := m.app.Restart(m.appContext(), newCfg)
		if err != nil {
			m.output.Errorf("Failed to restart server: %v", err)
		}
	})
	m.viper.WatchConfig()

	return m.versionCheckCmd()
}

func (m *uncorsApp) handleServerError(msg serverErrMsg) tea.Cmd {
	m.historyWidget.Update(outputLineMsg(msg.err.Error()))

	return tea.Quit
}

func (m *uncorsApp) handleRestart() {
	m.updateHistoryHeight()
}

func (m *uncorsApp) handleShutdown() tea.Cmd {
	_ = m.historyWidget.Close()

	return tea.Quit
}

func (m *uncorsApp) startServerCmd() tea.Cmd {
	return func() tea.Msg {
		err := m.app.Start(m.appContext(), m.cfg)
		if err != nil {
			return serverErrMsg{err: err}
		}

		return serverStartedMsg{}
	}
}

func (m *uncorsApp) waitOutputCmd() tea.Cmd {
	return func() tea.Msg {
		select {
		case line, ok := <-m.outputCh:
			if !ok {
				return nil
			}

			return outputLineMsg(line)
		case <-m.appDone:
			return nil
		}
	}
}

func (m *uncorsApp) watchEventsCmd() tea.Cmd {
	return func() tea.Msg {
		select {
		case event, ok := <-m.tracker.events:
			if !ok {
				return nil
			}

			return requestEventMsg(event)
		case <-m.appDone:
			return nil
		}
	}
}

func (m *uncorsApp) shutdownCmd() tea.Cmd {
	return func() tea.Msg {
		m.cancel()

		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		_ = m.app.Shutdown(ctx)

		return shutdownMsg{}
	}
}

func (m *uncorsApp) restartCmd() tea.Cmd {
	return func() tea.Msg {
		defer helpers.PanicInterceptor(func(value any) {
			m.output.Errorf("Restart error: %v", value)
		})

		newCfg := m.loadConfig()

		err := m.app.Restart(m.appContext(), newCfg)
		if err != nil {
			m.output.Errorf("Failed to restart: %v", err)
		}

		return restartMsg{}
	}
}

func (m *uncorsApp) versionCheckCmd() tea.Cmd {
	return func() tea.Msg {
		versionChecker := version.NewVersionChecker(
			version.WithOutput(m.output),
			version.WithHTTPClient(infra.MakeHTTPClient(m.cfg.Proxy)),
			version.WithCurrentVersion(m.version),
		)

		time.Sleep(versionCheckDelay)
		versionChecker.CheckNewVersion(m.appContext())

		return nil
	}
}
