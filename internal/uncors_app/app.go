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
	"github.com/evg4b/uncors/internal/tui"
)

const (
	outputChannelSize = 1000
	memTickInterval   = 2 * time.Second
	bytesPerMegabyte  = 1024 * 1024
)

type UncorsApp struct {
	keys keyMap

	// service owns the application runtime. The model only sends it commands
	// and renders what comes back.
	service service

	output   contracts.Output
	renderer *render.Renderer

	outputCh chan string
	done     <-chan struct{}

	termHeight int
	termWidth  int

	// historyHeight is the height last applied to the scrollback, so the
	// viewport is only resized when the footer actually changed size.
	historyHeight int

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

type (
	serviceEventMsg  struct{ event app.Event }
	serverStartedMsg struct{}
	serverErrMsg     struct{ err error }
	shutdownMsg      struct{}
	restartMsg       struct{}
)

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
	output := tui.NewCliOutput(newChannelWriter(outputCh))

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

	// A service event is rendered here and then re-dispatched as the message the
	// widgets speak, so a reload triggered by a config file save reaches them
	// exactly as the restart key does.
	if event, ok := msg.(serviceEventMsg); ok {
		m.renderer.Render(event.event)

		cmds = append(cmds, m.waitServiceEventCmd())

		if translated := widgetMessage(event.event); translated != nil {
			msg = translated
		}
	}

	switch typedMsg := msg.(type) {
	case tea.WindowSizeMsg:
		log.Printf("Window resized to %dx%d", typedMsg.Width, typedMsg.Height)
		m.termWidth = typedMsg.Width
		m.termHeight = typedMsg.Height

	case outputLineMsg:
		cmds = append(cmds, m.waitOutputCmd())

	case serverStartedMsg:
		// Watching the config file and checking for a new release belong to the
		// service, which started both as soon as the listeners were bound.
		log.Println("Server started")

	case serverErrMsg:
		cmds = append(cmds, m.handleServerError(typedMsg))

	case shutdownMsg:
		log.Println("Handling shutdown")

		_ = m.historyWidget.Close()

		return m, tea.Quit

	case tea.KeyPressMsg:
		log.Printf("Key pressed: %s", typedMsg.String())

		if cmd := m.handleKeyPress(typedMsg); cmd != nil {
			return m, cmd
		}
	}

	cmds = append(cmds, m.updateWidgets(msg)...)

	// Sized after the widgets have moved, so the scrollback is measured against
	// the footer as it is now rather than as it was.
	m.updateHistoryHeight()

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

// updateWidgets forwards msg to every widget and collects the work they ask for.
func (m *UncorsApp) updateWidgets(msg tea.Msg) []tea.Cmd {
	var cmd tea.Cmd

	cmds := make([]tea.Cmd, 0, 4) //nolint:mnd // one per widget below

	m.historyWidget, cmd = m.historyWidget.Update(msg)
	cmds = append(cmds, cmd)

	m.trackerWidget, cmd = m.trackerWidget.Update(msg)
	cmds = append(cmds, cmd)

	m.helpWidget, cmd = m.helpWidget.Update(msg)
	cmds = append(cmds, cmd)

	m.memWidget, cmd = m.memWidget.Update(msg)
	cmds = append(cmds, cmd)

	return cmds
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

// updateHistoryHeight gives the scrollback whatever the footer leaves it.
// Nothing is sized before the first WindowSizeMsg, because the terminal height
// is not known until then.
func (m *UncorsApp) updateHistoryHeight() {
	if m.termHeight == 0 {
		return
	}

	height := max(m.termHeight-m.footerHeight(), 1)
	if height == m.historyHeight {
		return
	}

	m.historyHeight = height
	m.historyWidget.SetHeight(height)
}

func (m *UncorsApp) footerHeight() int {
	footerHeight := m.helpWidget.Height()

	if m.trackerWidget.ActiveCount() > 0 {
		footerHeight += m.trackerWidget.Height()
	}

	return footerHeight
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
		// Deliberately not the service context: Shutdown cancels it, and the
		// shutdown has to outlive that. The grace period is the server's.
		_ = m.service.Shutdown(context.Background())
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
