package uncorsapp

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	help "charm.land/bubbles/v2/help"
	key "charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
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
	outputChannelSize = 100
	shutdownTimeout   = 5 * time.Second
	versionCheckDelay = 50 * time.Second
	tickInterval      = 200 * time.Millisecond
	memTickInterval   = 2 * time.Second
)

var (
	pendingMethodStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#8C8C8C")).
				Padding(0, 1)

	scrollBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555555"))

	memWidgetStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#777777"))
)

type uncorsApp struct {
	version string
	keys    keyMap
	help    help.Model

	app     *uncors.Uncors
	output  *tuiOutput
	tracker *requestTracker

	outputCh chan string
	ctx      context.Context
	cancel   context.CancelFunc

	cfg        *config.UncorsConfig
	loadConfig func() *config.UncorsConfig
	viper      *viper.Viper

	hist       *history
	vp         viewport.Model
	autoScroll bool
	termHeight int
	termWidth  int

	pending map[uint64]requestEvent
	ticking bool

	memMB float64
}

type (
	outputLineMsg    string
	serverStartedMsg struct{}
	serverErrMsg     struct{ err error }
	shutdownMsg      struct{}
	requestEventMsg  requestEvent
	tickMsg          struct{}
	restartMsg       struct{}
	memUpdateMsg     struct{ mb float64 }
)

func NewUncorsApp(
	ver string,
	fs afero.Fs,
	viperInstance *viper.Viper,
	cfg *config.UncorsConfig,
	loadConfig func() *config.UncorsConfig,
) tea.Model {
	ch := make(chan string, outputChannelSize)
	output := newTuiOutput(ch)
	tracker := newRequestTracker()
	ctx, cancel := context.WithCancel(context.Background())

	hist, err := newHistory()
	if err != nil {
		panic(fmt.Sprintf("failed to create history: %v", err))
	}

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	return uncorsApp{
		version: ver,
		keys:    newKeyMap(),
		help:    help.New(),
		app: uncors.CreateUncors(fs, output, ver).
			WithHandlerWrapper(tracker.Wrap),
		output:     output,
		tracker:    tracker,
		outputCh:   ch,
		ctx:        ctx,
		cancel:     cancel,
		cfg:        cfg,
		loadConfig: loadConfig,
		viper:      viperInstance,
		hist:       hist,
		vp:         viewport.New(),
		autoScroll: true,
		pending:    make(map[uint64]requestEvent),
		memMB:      float64(ms.HeapAlloc) / (1024 * 1024),
	}
}

func (m uncorsApp) Init() tea.Cmd {
	return tea.Batch(
		m.startServerCmd(),
		m.waitOutputCmd(),
		m.watchEventsCmd(),
		memTickCmd(),
	)
}

func (m uncorsApp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termHeight = msg.Height
		m.termWidth = msg.Width
		m.help.SetWidth(msg.Width)
		m.vp.SetWidth(msg.Width)
		m.vp.SetHeight(m.historyHeight())

		if m.autoScroll {
			m.vp.GotoBottom()
		}

	case outputLineMsg:
		atBottom := m.autoScroll
		m.hist.AppendLine(string(msg))
		m.vp.SetHeight(m.historyHeight())
		m.vp.SetContentLines(m.hist.Lines())

		if atBottom {
			m.vp.GotoBottom()
		}

		return m, m.waitOutputCmd()

	case requestEventMsg:
		if msg.done {
			delete(m.pending, msg.id)
		} else {
			m.pending[msg.id] = requestEvent(msg)
		}

		m.vp.SetHeight(m.historyHeight())

		if m.autoScroll {
			m.vp.GotoBottom()
		}

		cmds := []tea.Cmd{m.watchEventsCmd()}
		if len(m.pending) > 0 && !m.ticking {
			m.ticking = true
			cmds = append(cmds, m.tickCmd())
		}

		return m, tea.Batch(cmds...)

	case tickMsg:
		if len(m.pending) > 0 {
			return m, m.tickCmd()
		}

		m.ticking = false

	case memUpdateMsg:
		m.memMB = msg.mb

		return m, memTickCmd()

	case serverStartedMsg:
		m.viper.OnConfigChange(func(_ fsnotify.Event) {
			defer helpers.PanicInterceptor(func(value any) {
				m.output.Errorf("Config reloading error: %v", value)
			})

			newCfg := m.loadConfig()
			err := m.app.Restart(m.ctx, newCfg)
			if err != nil {
				m.output.Errorf("Failed to restart server: %v", err)
			}
		})
		m.viper.WatchConfig()

		return m, m.versionCheckCmd()

	case serverErrMsg:
		m.hist.AppendLine(msg.err.Error())
		m.vp.SetContentLines(m.hist.Lines())
		m.vp.GotoBottom()

		return m, tea.Quit

	case restartMsg:
		m.pending = make(map[uint64]requestEvent)
		m.ticking = false
		m.vp.SetHeight(m.historyHeight())

	case shutdownMsg:
		_ = m.hist.Close()

		return m, tea.Quit

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
			m.vp.SetHeight(m.historyHeight())
		case key.Matches(msg, m.keys.ScrollUp):
			m.vp.ScrollUp(1)
			m.autoScroll = m.vp.AtBottom()
		case key.Matches(msg, m.keys.ScrollDown):
			m.vp.ScrollDown(1)
			m.autoScroll = m.vp.AtBottom()
		case key.Matches(msg, m.keys.PageUp):
			m.vp.PageUp()
			m.autoScroll = m.vp.AtBottom()
		case key.Matches(msg, m.keys.PageDown):
			m.vp.PageDown()
			m.autoScroll = m.vp.AtBottom()
		case key.Matches(msg, m.keys.GotoTop):
			m.vp.GotoTop()
			m.autoScroll = false
		case key.Matches(msg, m.keys.GotoBottom):
			m.vp.GotoBottom()
			m.autoScroll = true
		case key.Matches(msg, m.keys.Restart):
			return m, m.restartCmd()
		case key.Matches(msg, m.keys.Quit):
			return m, m.shutdownCmd()
		}
	}

	return m, nil
}

func (m uncorsApp) View() tea.View {
	var b strings.Builder

	b.WriteString(m.vp.View())
	b.WriteByte('\n')

	if m.hist.LineCount() > 0 {
		b.WriteString(m.renderStatusBar())
		b.WriteByte('\n')
	}

	if len(m.pending) > 0 {
		fmt.Fprintf(&b, "In progress (%d):\n", len(m.pending))

		for _, req := range m.pending {
			elapsed := formatElapsed(time.Since(req.startedAt))
			fmt.Fprintf(&b, "  %s %s  %s\n",
				pendingMethodStyle.Render(req.method),
				req.url.String(),
				elapsed,
			)
		}
	}

	b.WriteString(m.renderHelpBar())

	v := tea.NewView(b.String())
	v.AltScreen = true

	return v
}

// historyHeight returns the number of lines the viewport should occupy.
func (m uncorsApp) historyHeight() int {
	h := m.termHeight - m.footerHeight()
	if h < 1 {
		return 1
	}

	return h
}

// footerHeight returns the total number of lines rendered below the viewport.
func (m uncorsApp) footerHeight() int {
	h := 1 // help bar
	if m.hist.LineCount() > 0 {
		h++ // status bar — only when there is something to scroll
	}

	if len(m.pending) > 0 {
		h += 1 + len(m.pending) // "In progress:" header + N request lines
	}

	if m.help.ShowAll {
		h += 2 // FullHelp has 3 rows; +2 beyond the base row
	}

	return h
}

func (m uncorsApp) renderStatusBar() string {
	pct := int(m.vp.ScrollPercent() * 100) //nolint:mnd

	scrollStr := fmt.Sprintf("%d%%", pct)
	if m.autoScroll {
		scrollStr += " [auto]"
	}

	left := scrollBarStyle.Render(fmt.Sprintf("─ %s (%d lines) ", scrollStr, m.hist.LineCount()))
	fill := scrollBarStyle.Render(strings.Repeat("─", max(0, m.termWidth-lipgloss.Width(left))))

	return left + fill
}

func (m uncorsApp) renderHelpBar() string {
	helpStr := m.help.View(m.keys)
	memStr := memWidgetStyle.Render(fmt.Sprintf("[ %.1f MB ]", m.memMB))

	gap := m.termWidth - lipgloss.Width(helpStr) - lipgloss.Width(memStr)
	if gap > 0 {
		return helpStr + strings.Repeat(" ", gap) + memStr
	}

	return helpStr
}

func (m uncorsApp) startServerCmd() tea.Cmd {
	return func() tea.Msg {
		err := m.app.Start(m.ctx, m.cfg)
		if err != nil {
			return serverErrMsg{err: err}
		}

		return serverStartedMsg{}
	}
}

func (m uncorsApp) waitOutputCmd() tea.Cmd {
	return func() tea.Msg {
		select {
		case line, ok := <-m.outputCh:
			if !ok {
				return nil
			}

			return outputLineMsg(line)
		case <-m.ctx.Done():
			return nil
		}
	}
}

func (m uncorsApp) watchEventsCmd() tea.Cmd {
	return func() tea.Msg {
		select {
		case event, ok := <-m.tracker.events:
			if !ok {
				return nil
			}

			return requestEventMsg(event)
		case <-m.ctx.Done():
			return nil
		}
	}
}

func (m uncorsApp) tickCmd() tea.Cmd {
	return tea.Tick(tickInterval, func(_ time.Time) tea.Msg {
		return tickMsg{}
	})
}

func memTickCmd() tea.Cmd {
	return tea.Tick(memTickInterval, func(_ time.Time) tea.Msg {
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)

		return memUpdateMsg{mb: float64(ms.HeapAlloc) / (1024 * 1024)}
	})
}

func (m uncorsApp) shutdownCmd() tea.Cmd {
	return func() tea.Msg {
		m.cancel()

		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		_ = m.app.Shutdown(ctx)

		return shutdownMsg{}
	}
}

func (m uncorsApp) restartCmd() tea.Cmd {
	return func() tea.Msg {
		defer helpers.PanicInterceptor(func(value any) {
			m.output.Errorf("Restart error: %v", value)
		})

		newCfg := m.loadConfig()
		err := m.app.Restart(m.ctx, newCfg)
		if err != nil {
			m.output.Errorf("Failed to restart: %v", err)
		}

		return restartMsg{}
	}
}

func (m uncorsApp) versionCheckCmd() tea.Cmd {
	return func() tea.Msg {
		versionChecker := version.NewVersionChecker(
			version.WithOutput(m.output),
			version.WithHTTPClient(infra.MakeHTTPClient(m.cfg.Proxy)),
			version.WithCurrentVersion(m.version),
		)

		time.Sleep(versionCheckDelay)
		versionChecker.CheckNewVersion(m.ctx)

		return nil
	}
}

func formatElapsed(d time.Duration) string {
	d = d.Truncate(time.Millisecond)
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}

	return fmt.Sprintf("%.1fs", d.Seconds())
}
