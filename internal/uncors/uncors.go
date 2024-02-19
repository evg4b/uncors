package uncors

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/spf13/afero"
)

type uncorsModel struct {
	logPrinter     *tui.Printer
	version        string
	keys           keyMap
	help           help.Model
	config         *config.UncorsConfig
	requestTracker tui.RequestTracker
	app            *App
	spinner        spinner.Model
	width          int
	configLoader   *tui.ConfigLoader
}

// keyMap defines a set of keybindings. To work for help it must satisfy
// key.Map. It could also very easily be a map[string]key.Binding.
type keyMap struct {
	Quit    key.Binding
	Restart key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view. It's part
// of the key.Map interface.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Quit,
		k.Restart,
	}
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Quit, k.Restart}, // second column
	}
}

var keys = keyMap{
	Restart: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "restart server")),
	Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}

type Option = func(*uncorsModel)

func NewUncorsModel(options ...Option) tea.Model {
	model := uncorsModel{
		keys:    keys,
		help:    help.New(),
		spinner: spinner.New(spinner.WithSpinner(tui.Spinner)),
	}
	helpers.ApplyOptions(&model, options)
	model.app = CreateApp(afero.NewOsFs(), model.version, model.requestTracker)

	return model
}

func (u uncorsModel) Init() tea.Cmd {
	return tea.Batch(
		u.configLoader.Init,
		u.logPrinter.Tick,
		u.requestTracker.Tick,
		u.requestTracker.Tick2,
		u.spinner.Tick,
		tea.HideCursor,
		tea.SetWindowTitle(fmt.Sprintf("uncors v%s", u.version)),
		tea.Sequence(
			tui.PrintLogoCmd(u.version),
			tea.Println(),
			tui.PrintDisclaimerMessage(),
			tea.Println(),
			tui.PrintMappings(u.config.Mappings),
		),
		func() tea.Msg {
			u.app.Start(context.Background(), u.config)

			return nil
		},
	)
}

func (u uncorsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tui.PrinterMsg:
		return u, u.logPrinter.Update(msg)
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, u.keys.Restart):
			return u, tea.Batch(
				func() tea.Msg {
					u.app.Restart(context.Background(), u.config)

					return nil
				},
				tea.ClearScreen,
			)
		case key.Matches(msg, u.keys.Quit):
			if err := u.app.Shutdown(context.Background()); err != nil {
				log.Error(err)
			}

			return u, tea.Quit
		}
	case *config.UncorsConfig:
		u.config = msg
		u.app.Restart(context.Background(), u.config)
		return u, tea.Batch(
			u.configLoader.Tick,
			tea.Sequence(
				tea.Println("config updated"),
				tea.ClearScreen,
				tui.PrintMappings(u.config.Mappings),
			),
		)
	case tea.WindowSizeMsg:
		u.width = msg.Width
		u.help.Width = msg.Width

		return u, nil
	case tui.DoneRequestDefinition:
		return u, tea.Batch(
			u.requestTracker.Tick,
			tea.Println(tui.RenderDoneRequest(msg)),
		)
	case tui.RequestDefinition:
		return u, u.requestTracker.Tick2
	case spinner.TickMsg:
		var cmd tea.Cmd
		u.spinner, cmd = u.spinner.Update(msg)

		return u, cmd
	}

	return u, nil
}

func (u uncorsModel) View() string {
	data := u.requestTracker.View(u.spinner.View())

	if data == "" {
		return u.help.View(u.keys)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		data,
		u.help.View(u.keys),
	)
}
