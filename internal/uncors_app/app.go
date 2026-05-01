package uncorsapp

import (
	"context"
	"time"

	help "charm.land/bubbles/v2/help"
	key "charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
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
	outputChannelSize  = 100
	shutdownTimeout    = 5 * time.Second
	versionCheckDelay  = 50 * time.Second
)

type uncorsApp struct {
	version string
	keys    keyMap
	help    help.Model

	app    *uncors.Uncors
	output *tuiOutput

	outputCh chan string
	ctx      context.Context
	cancel   context.CancelFunc

	cfg        *config.UncorsConfig
	loadConfig func() *config.UncorsConfig
	viper      *viper.Viper
}

type outputLineMsg string
type serverStartedMsg struct{}
type serverErrMsg struct{ err error }
type shutdownMsg struct{}

func NewUncorsApp(
	ver string,
	fs afero.Fs,
	viperInstance *viper.Viper,
	cfg *config.UncorsConfig,
	loadConfig func() *config.UncorsConfig,
) tea.Model {
	ch := make(chan string, outputChannelSize)
	output := newTuiOutput(ch)
	ctx, cancel := context.WithCancel(context.Background())

	return uncorsApp{
		version:    ver,
		keys:       newKeyMap(),
		help:       help.New(),
		app:        uncors.CreateUncors(fs, output, ver),
		output:     output,
		outputCh:   ch,
		ctx:        ctx,
		cancel:     cancel,
		cfg:        cfg,
		loadConfig: loadConfig,
		viper:      viperInstance,
	}
}

func (m uncorsApp) Init() tea.Cmd {
	return tea.Sequence(
		tea.ClearScreen,
		tea.Batch(
			m.startServerCmd(),
			m.waitOutputCmd(),
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

	case shutdownMsg:
		return m, tea.Quit

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
		case key.Matches(msg, m.keys.Quit):
			return m, m.shutdownCmd()
		}
	}

	return m, nil
}

func (m uncorsApp) View() tea.View {
	return tea.NewView(m.help.View(m.keys))
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

func (m uncorsApp) shutdownCmd() tea.Cmd {
	return func() tea.Msg {
		m.cancel()
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		_ = m.app.Shutdown(ctx)
		return shutdownMsg{}
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
