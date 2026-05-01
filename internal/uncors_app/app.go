package uncorsapp

import (
	"context"
	"fmt"
	"strings"
	"time"

	help "charm.land/bubbles/v2/help"
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
	outputChannelSize = 100
	shutdownTimeout   = 5 * time.Second
	versionCheckDelay = 50 * time.Second
	tickInterval      = 200 * time.Millisecond
)

var pendingMethodStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#FFFFFF")).
	Background(lipgloss.Color("#8C8C8C")).
	Padding(0, 1)

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

	pending map[uint64]requestEvent
	ticking bool
}

type outputLineMsg string
type serverStartedMsg struct{}
type serverErrMsg struct{ err error }
type shutdownMsg struct{}
type requestEventMsg requestEvent
type tickMsg struct{}
type restartMsg struct{}

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
		pending:    make(map[uint64]requestEvent),
	}
}

func (m uncorsApp) Init() tea.Cmd {
	return tea.Sequence(
		tea.ClearScreen,
		tea.Batch(
			m.startServerCmd(),
			m.waitOutputCmd(),
			m.watchEventsCmd(),
		),
	)
}

func (m uncorsApp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.help.SetWidth(msg.Width)

	case outputLineMsg:
		return m, tea.Sequence(
			tea.Println(string(msg)),
			m.waitOutputCmd(),
		)

	case requestEventMsg:
		if msg.done {
			delete(m.pending, msg.id)
		} else {
			m.pending[msg.id] = requestEvent(msg)
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
		return m, nil

	case serverStartedMsg:
		m.viper.OnConfigChange(func(_ fsnotify.Event) {
			defer helpers.PanicInterceptor(func(value any) {
				m.output.Errorf("Config reloading error: %v", value)
			})
			newCfg := m.loadConfig()
			if err := m.app.Restart(m.ctx, newCfg); err != nil {
				m.output.Errorf("Failed to restart server: %v", err)
			}
		})
		m.viper.WatchConfig()
		return m, m.versionCheckCmd()

	case serverErrMsg:
		return m, tea.Sequence(
			tea.Println(msg.err.Error()),
			tea.Quit,
		)

	case restartMsg:
		m.pending = make(map[uint64]requestEvent)
		m.ticking = false
		return m, nil

	case shutdownMsg:
		return m, tea.Quit

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
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
		b.WriteByte('\n')
	}

	b.WriteString(m.help.View(m.keys))

	return tea.NewView(b.String())
}

func (m uncorsApp) startServerCmd() tea.Cmd {
	return func() tea.Msg {
		if err := m.app.Start(m.ctx, m.cfg); err != nil {
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
		if err := m.app.Restart(m.ctx, newCfg); err != nil {
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
