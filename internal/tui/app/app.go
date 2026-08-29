package app

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/di"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/internal/uncors"
)

const (
	outputChannelSize = 1000
	memTickInterval   = 2 * time.Second
	bytesPerMegabyte  = 1024 * 1024
)

type UncorsApp struct {
	keys keyMap

	app     *uncors.Uncors
	output  *Output
	tracker server.IRequestTracker

	outputCh   <-chan string
	logCh      <-chan string
	appContext func() context.Context
	appDone    <-chan struct{}
	cancel     context.CancelFunc

	cfg *config.UncorsConfig

	reloader *uncors.Reloader
	runner   *uncors.Runner

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
	update(app *UncorsApp) tea.Cmd
}

// NewUncorsApp creates the interactive TUI model. configPath is the active
// config file path (empty string if no config file is used); when non-empty
// the app watches it for changes and auto-restarts the proxy on every save.
func NewUncorsApp(
	container *di.Container,
	output *Output,
	configPath string,
	cfg *config.UncorsConfig,
	loadConfig uncors.ConfigLoader,
	checkVersion uncors.VersionCheck,
) *UncorsApp {
	appCtx, cancel := context.WithCancel(context.Background())

	keys := newKeyMap()

	historyWidget := NewHistoryWidget(keys)

	app := uncors.CreateUncors(container)
	reloader := uncors.NewReloader(app, output, loadConfig, configPath)

	return &UncorsApp{
		keys:          keys,
		app:           app,
		output:        output,
		tracker:       container.RequestTracker(),
		outputCh:      output.Lines(),
		logCh:         logRecords(),
		appContext:    func() context.Context { return appCtx },
		appDone:       appCtx.Done(),
		cancel:        cancel,
		cfg:           cfg,
		reloader:      reloader,
		runner:        uncors.NewRunner(app, reloader, output, checkVersion),
		historyWidget: historyWidget,
		trackerWidget: NewTrackerWidget(),
		helpWidget:    NewHelpWidget(keys),
		memWidget:     NewMemoryWidget(),
	}
}

func (m *UncorsApp) Init() tea.Cmd {
	slog.Debug("Initializing UncorsApp")

	return tea.Batch(
		m.startServerCmd(),
		m.waitOutputCmd(),
		m.waitLogCmd(),
		m.watchEventsCmd(),
		m.memWidget.Init(),
		m.trackerWidget.Init(),
		m.historyWidget.Init(),
		m.helpWidget.Init(),
	)
}

func (m *UncorsApp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch typedMsg := msg.(type) {
	case tea.WindowSizeMsg:
		slogDebugf("Window resized to %dx%d", typedMsg.Width, typedMsg.Height)
		m.termHeight = typedMsg.Height
		m.termWidth = typedMsg.Width
		m.updateHistoryHeight()

	case restartMsg:
		slog.Debug("Restart message received")
		m.handleRestart()

	case outputLineMsg:
		cmds = append(cmds, m.waitOutputCmd())

	case logLineMsg:
		m.historyWidget.Update(outputLineMsg(typedMsg))

		cmds = append(cmds, m.waitLogCmd())

	case requestEventMsg:
		m.handleRequestEvent(typedMsg)

		cmds = append(cmds, m.watchEventsCmd())

	case tea.KeyPressMsg:
		slogDebugf("Key pressed: %s", typedMsg.String())

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

func (m *UncorsApp) View() tea.View {
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

func (m *UncorsApp) updateLayout(msg tea.Msg) {
	if _, isRequest := msg.(requestEventMsg); isRequest {
		m.updateHistoryHeight()
	} else if _, isKey := msg.(tea.KeyPressMsg); isKey {
		m.updateHistoryHeight()
	}
}

func (m *UncorsApp) handleKeyPress(msg tea.KeyPressMsg) tea.Cmd {
	if key.Matches(msg, m.keys.Restart) {
		return m.restartCmd()
	}

	if key.Matches(msg, m.keys.Quit) {
		return m.shutdownCmd()
	}

	return nil
}

func (m *UncorsApp) updateHistoryHeight() {
	footerHeight := m.footerHeight()
	viewportHeight := max(m.termHeight-footerHeight, 1)

	slogDebugf(
		"Updating layout: termHeight=%d, footerHeight=%d => viewportHeight=%d",
		m.termHeight,
		footerHeight,
		viewportHeight,
	)
	m.historyWidget.SetHeight(viewportHeight)
}

func (m *UncorsApp) footerHeight() int {
	footerHeight := m.helpWidget.Height()

	if m.trackerWidget.ActiveCount() > 0 {
		footerHeight += m.trackerWidget.Height()
	}

	return footerHeight
}

// The runner starts config watching, signal handling and the version check as
// part of startup, so there is nothing left for the model to kick off here.
func (msg serverStartedMsg) update(*UncorsApp) tea.Cmd {
	return nil
}

func (msg serverErrMsg) update(app *UncorsApp) tea.Cmd {
	return app.handleServerError(msg)
}

func (msg shutdownMsg) update(app *UncorsApp) tea.Cmd {
	return app.handleShutdown()
}

func (m *UncorsApp) handleServerError(msg serverErrMsg) tea.Cmd {
	m.historyWidget.Update(outputLineMsg(msg.err.Error()))

	return tea.Quit
}

func (m *UncorsApp) handleRequestEvent(event requestEventMsg) {
	if !event.Done || event.Data == nil {
		return
	}

	if event.Prefix != "" {
		m.output.NewPrefixOutput(event.Prefix).Request(event.Data)
	} else {
		m.output.Request(event.Data)
	}
}

func (m *UncorsApp) handleRestart() {
	slog.Debug("Handling restart")
	m.updateHistoryHeight()
}

func (m *UncorsApp) handleShutdown() tea.Cmd {
	slog.Debug("Handling shutdown")

	_ = m.reloader.Close()

	_ = m.historyWidget.Close()

	return tea.Quit
}

func (m *UncorsApp) startServerCmd() tea.Cmd {
	return func() tea.Msg {
		err := m.runner.Start(m.appContext(), m.cfg)
		if err != nil {
			return serverErrMsg{err: err}
		}

		return serverStartedMsg{}
	}
}

func (m *UncorsApp) waitOutputCmd() tea.Cmd {
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

// waitLogCmd renders the process diagnostics in the history view, which is
// where they have to go in interactive mode: writing them to the terminal would
// corrupt the alt-screen.
func (m *UncorsApp) waitLogCmd() tea.Cmd {
	if m.logCh == nil {
		return nil
	}

	return func() tea.Msg {
		select {
		case line, ok := <-m.logCh:
			if !ok {
				return nil
			}

			return logLineMsg(line)
		case <-m.appDone:
			return nil
		}
	}
}

func (m *UncorsApp) watchEventsCmd() tea.Cmd {
	return func() tea.Msg {
		select {
		case event, ok := <-m.tracker.Events():
			if !ok {
				return nil
			}

			return requestEventMsg(event)
		case <-m.appDone:
			return nil
		}
	}
}

func (m *UncorsApp) shutdownCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := m.appContext()
		m.cancel()

		err := m.runner.Shutdown(ctx)
		if err != nil {
			slog.Error("shutdown failed", "err", err)
		}

		return shutdownMsg{}
	}
}

func (m *UncorsApp) restartCmd() tea.Cmd {
	return func() tea.Msg {
		err := m.runner.Reload(m.appContext())
		if err != nil {
			slog.Error("restart failed", "err", err)
			m.output.Errorf("Failed to restart: %v", err)
		}

		return restartMsg{}
	}
}
