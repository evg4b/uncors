package uncorsapp

import (
	"context"
	"log"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/evg4b/uncors/internal/app"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/di"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/render"
)

const (
	outputChannelSize = 1000
	shutdownTimeout   = 15 * time.Second
	memTickInterval   = 2 * time.Second
	bytesPerMegabyte  = 1024 * 1024
)

type UncorsApp struct {
	keys keyMap

	// service owns the application runtime. The model only sends it commands
	// and renders what comes back.
	service service

	output   *tuiOutput
	renderer *render.Renderer

	outputCh chan string
	done     <-chan struct{}

	termHeight int
	termWidth  int

	historyWidget *HistoryWidget
	trackerWidget *TrackerWidget
	helpWidget    *HelpWidget
	memWidget     *MemoryWidget
}

// service is the whole of the model's dependency on the application. Commands
// go down (Start, Reload, Shutdown), events come back up; the model reaches
// for nothing else, which is what keeps application behaviour out of the TUI.
type service interface {
	Start(ctx context.Context) error
	Reload()
	Shutdown(ctx context.Context) error
	Close() error
	Context() context.Context
	Events() <-chan app.Event
}

type serviceEventMsg struct{ event app.Event }

type (
	serverStartedMsg struct{}
	serverErrMsg     struct{ err error }
	shutdownMsg      struct{}
	restartMsg       struct{}
)

type appUpdateMsg interface {
	update(app *UncorsApp) tea.Cmd
}

// NewUncorsApp creates the interactive TUI model over the application service.
// configPath is the active config file path (empty when no config file is in
// use); the service watches it and reloads on every save.
func NewUncorsApp(
	container *di.Container,
	configPath string,
	cfg *config.UncorsConfig,
	loadConfig app.Loader,
) *UncorsApp {
	outputCh := make(chan string, outputChannelSize)
	output := newTuiOutput(outputCh)

	// The sink has to be installed before anything resolves CliOutput, because
	// the container caches it on first use.
	container.Override(di.WithCliOutput(func() contracts.Output {
		return output
	}))

	service := app.New(container, cfg, configPath, loadConfig)

	keys := newKeyMap()

	return &UncorsApp{
		keys:          keys,
		service:       service,
		output:        output,
		renderer:      render.New(output, container.Version()),
		outputCh:      outputCh,
		done:          service.Context().Done(),
		historyWidget: NewHistoryWidget(keys),
		trackerWidget: NewTrackerWidget(),
		helpWidget:    NewHelpWidget(keys),
		memWidget:     NewMemoryWidget(),
	}
}

func (m *UncorsApp) Init() tea.Cmd {
	log.Println("Initializing UncorsApp")

	return tea.Batch(
		m.startServerCmd(),
		m.waitOutputCmd(),
		m.waitServiceEventCmd(),
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
		log.Printf("Window resized to %dx%d", typedMsg.Width, typedMsg.Height)
		m.termHeight = typedMsg.Height
		m.termWidth = typedMsg.Width
		m.updateHistoryHeight()

	case restartMsg:
		log.Println("Restart message received")
		m.handleRestart()

	case outputLineMsg:
		cmds = append(cmds, m.waitOutputCmd())

	case serviceEventMsg:
		m.renderer.Render(typedMsg.event)

		cmds = append(cmds, m.waitServiceEventCmd())

		// Widgets react to the service's own account of what happened, so a
		// reload triggered by a file save reaches them exactly as the restart
		// key does.
		if translated := widgetMessage(typedMsg.event); translated != nil {
			msg = translated
		}

	case tea.KeyPressMsg:
		log.Printf("Key pressed: %s", typedMsg.String())

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

	log.Printf(
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

func (msg serverStartedMsg) update(app *UncorsApp) tea.Cmd {
	return app.handleServerStarted()
}

func (msg serverErrMsg) update(app *UncorsApp) tea.Cmd {
	return app.handleServerError(msg)
}

func (msg shutdownMsg) update(app *UncorsApp) tea.Cmd {
	return app.handleShutdown()
}

// handleServerStarted runs once the listeners are bound. Config watching and
// the version check belong to the service, which started them itself.
func (m *UncorsApp) handleServerStarted() tea.Cmd {
	log.Println("Server started")

	return nil
}

func (m *UncorsApp) handleServerError(msg serverErrMsg) tea.Cmd {
	m.historyWidget, _ = m.historyWidget.Update(outputLineMsg(msg.err.Error()))

	// Quitting straight away would strand the generation the failed start left
	// behind, so go through the normal shutdown path instead.
	return m.shutdownCmd()
}

// widgetMessage translates a service event into the message the widgets speak,
// or nil when no widget cares about it.
func widgetMessage(event app.Event) tea.Msg {
	switch typed := event.(type) {
	case app.RequestEvent:
		return requestEventMsg(typed.Event)
	case app.LifecycleEvent:
		if typed.State == app.StateReloaded {
			return restartMsg{}
		}
	case app.LogEvent:
	}

	return nil
}

func (m *UncorsApp) handleRestart() {
	log.Println("Handling restart")
	m.updateHistoryHeight()
}

func (m *UncorsApp) handleShutdown() tea.Cmd {
	log.Println("Handling shutdown")

	_ = m.historyWidget.Close()

	return tea.Quit
}

func (m *UncorsApp) startServerCmd() tea.Cmd {
	return func() tea.Msg {
		err := m.service.Start(m.service.Context())
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
		case <-m.done:
			return nil
		}
	}
}

// waitServiceEventCmd pulls one service event and renders it into the history.
// Re-armed on every serviceEventMsg, the way the other stream readers are.
func (m *UncorsApp) waitServiceEventCmd() tea.Cmd {
	return func() tea.Msg {
		select {
		case event, ok := <-m.service.Events():
			if !ok {
				return nil
			}

			return serviceEventMsg{event: event}
		case <-m.done:
			return nil
		}
	}
}

func (m *UncorsApp) shutdownCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		_ = m.service.Shutdown(ctx)
		_ = m.service.Close()

		return shutdownMsg{}
	}
}

func (m *UncorsApp) restartCmd() tea.Cmd {
	return func() tea.Msg {
		defer helpers.PanicInterceptor(func(value any) {
			m.output.Errorf("Restart error: %v", value)
		})

		// The reload's effects arrive as service events, which is what the
		// widgets act on; nothing to report from here.
		m.service.Reload()

		return nil
	}
}
