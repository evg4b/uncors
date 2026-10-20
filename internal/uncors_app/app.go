package uncorsapp

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/telemetry"
	"github.com/evg4b/uncors/internal/tui"
)

const (
	memTickInterval  = 2 * time.Second
	bytesPerMegabyte = 1024 * 1024
)

type UncorsApp struct {
	keys KeyMap

	events   *events.Subscription
	commands chan<- events.Command
	// stopped is closed once the proxy has stopped, so a command that can no
	// longer be delivered does not block forever.
	stopped chan struct{}

	rendered *bytes.Buffer
	renderer *tui.EventRenderer

	termHeight int
	termWidth  int

	historyWidget *HistoryWidget
	trackerWidget *TrackerWidget
	helpWidget    *HelpWidget
	memWidget     *MemoryWidget
}

type (
	lifecycleMsg    struct{ event events.Event }
	eventsClosedMsg struct{}
)

type AppOption = func(*UncorsApp)

func WithMemoryWidget(widget *MemoryWidget) AppOption {
	return func(app *UncorsApp) {
		app.memWidget = widget
	}
}

func NewUncorsApp(
	subscription *events.Subscription,
	commands chan<- events.Command,
	version string,
	options ...AppOption,
) *UncorsApp {
	keys := NewKeyMap()
	rendered := &bytes.Buffer{}

	return helpers.ApplyOptions(&UncorsApp{
		keys:          keys,
		events:        subscription,
		commands:      commands,
		stopped:       make(chan struct{}),
		rendered:      rendered,
		renderer:      tui.NewEventRenderer(tui.NewCliOutput(rendered), version),
		historyWidget: NewHistoryWidget(keys),
		trackerWidget: NewTrackerWidget(),
		helpWidget:    NewHelpWidget(keys),
		memWidget:     NewMemoryWidget(),
	}, options)
}

func (m *UncorsApp) Init() tea.Cmd {
	telemetry.Debug(context.Background(), "Initializing UncorsApp")

	return tea.Batch(
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
		m.resize(typedMsg)

	case requestStartedMsg:
		cmds = append(cmds, m.watchEventsCmd())

	case requestCompletedMsg:
		m.render(events.RequestCompleted(typedMsg))

		cmds = append(cmds, m.watchEventsCmd())

	case lifecycleMsg:
		m.render(typedMsg.event)

		if _, stopped := typedMsg.event.(events.ServerStopped); stopped {
			return m, m.quit()
		}

		cmds = append(cmds, m.watchEventsCmd())

	case eventsClosedMsg:
		return m, m.quit()

	case tea.KeyPressMsg:
		if cmd := m.handleKeyPress(typedMsg); cmd != nil {
			return m, cmd
		}
	}

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

	m.updateLayout(msg)

	return m, tea.Batch(cmds...)
}

func (m *UncorsApp) View() tea.View {
	var viewBuilder strings.Builder

	viewBuilder.WriteString(m.historyWidget.View().Content)

	if m.trackerWidget.ActiveCount() > 0 {
		viewBuilder.WriteByte('\n')
		viewBuilder.WriteString(m.trackerWidget.View().Content)
	}

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
	switch msg.(type) {
	case requestStartedMsg, requestCompletedMsg, lifecycleMsg, tea.KeyPressMsg:
		m.updateHistoryHeight()
	}
}

func (m *UncorsApp) resize(msg tea.WindowSizeMsg) {
	if telemetry.Enabled() {
		telemetry.Debug(context.Background(), "window resized",
			slog.Int("width", msg.Width), slog.Int("height", msg.Height))
	}

	m.termHeight = msg.Height
	m.termWidth = msg.Width
	m.updateHistoryHeight()
}

func (m *UncorsApp) handleKeyPress(msg tea.KeyPressMsg) tea.Cmd {
	if telemetry.Enabled() {
		telemetry.Debug(context.Background(), "key pressed", slog.String("key", msg.String()))
	}

	if key.Matches(msg, m.keys.Restart) {
		return m.sendCmd(events.Restart{})
	}

	if key.Matches(msg, m.keys.Quit) {
		return m.sendCmd(events.Shutdown{})
	}

	return nil
}

func (m *UncorsApp) updateHistoryHeight() {
	footerHeight := m.footerHeight()
	viewportHeight := max(m.termHeight-footerHeight, 1)

	if telemetry.Enabled() {
		telemetry.Debug(context.Background(), "updating layout",
			slog.Int("termHeight", m.termHeight),
			slog.Int("footerHeight", footerHeight),
			slog.Int("viewportHeight", viewportHeight),
		)
	}

	m.historyWidget.SetHeight(viewportHeight)
}

func (m *UncorsApp) footerHeight() int {
	footerHeight := m.helpWidget.Height()

	if m.trackerWidget.ActiveCount() > 0 {
		footerHeight += m.trackerWidget.Height()
	}

	return footerHeight
}

func (m *UncorsApp) render(event events.Event) {
	m.renderer.Render(event)

	text := strings.TrimRight(m.rendered.String(), "\n")
	m.rendered.Reset()

	if text != "" {
		m.historyWidget.AppendLine(text)
	}
}

func (m *UncorsApp) quit() tea.Cmd {
	telemetry.Debug(context.Background(), "Handling shutdown")

	select {
	case <-m.stopped:
	default:
		close(m.stopped)
	}

	_ = m.historyWidget.Close()

	return tea.Quit
}

func (m *UncorsApp) sendCmd(command events.Command) tea.Cmd {
	return func() tea.Msg {
		select {
		case m.commands <- command:
		case <-m.stopped:
		}

		return nil
	}
}

func (m *UncorsApp) watchEventsCmd() tea.Cmd {
	return func() tea.Msg {
		event, ok := <-m.events.Events()
		if !ok {
			return eventsClosedMsg{}
		}

		return EventMsg(event)
	}
}

func EventMsg(event events.Event) tea.Msg {
	switch typed := event.(type) {
	case events.RequestStarted:
		return requestStartedMsg(typed)
	case events.RequestCompleted:
		return requestCompletedMsg(typed)
	default:
		return lifecycleMsg{event: event}
	}
}
