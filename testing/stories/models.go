package stories

import (
	tea "charm.land/bubbletea/v2"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/tui"
)

const commandBuffer = 16

type widget[T any] interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (T, tea.Cmd)
	View() tea.View
}

// widgetModel adapts a widget, whose Update returns its own type, to tea.Model.
type widgetModel[T widget[T]] struct {
	widget T
	resize func(widget T, size tea.WindowSizeMsg)
}

func (m *widgetModel[T]) Init() tea.Cmd {
	return m.widget.Init()
}

func (m *widgetModel[T]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok && m.resize != nil {
		m.resize(m.widget, size)
	}

	var cmd tea.Cmd

	m.widget, cmd = m.widget.Update(msg)

	return m, cmd
}

func (m *widgetModel[T]) View() tea.View {
	return m.widget.View()
}

// replayModel applies the story's messages up front and runs the commands
// they returned from Init, so spinners and tickers keep going in tuiplay.
type replayModel struct {
	tea.Model

	cmds []tea.Cmd
}

func replay(model tea.Model, msgs ...tea.Msg) *replayModel {
	replayed := &replayModel{Model: model}

	for _, msg := range msgs {
		var cmd tea.Cmd

		replayed.Model, cmd = replayed.Model.Update(msg)
		replayed.cmds = append(replayed.cmds, cmd)
	}

	return replayed
}

func (m *replayModel) Init() tea.Cmd {
	return tea.Batch(append(m.cmds, m.Model.Init())...)
}

func (m *replayModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	m.Model, cmd = m.Model.Update(msg)

	return m, cmd
}

func widgetStory[T widget[T]](view T, resize func(T, tea.WindowSizeMsg), msgs ...tea.Msg) tea.Model {
	return replay(&widgetModel[T]{widget: view, resize: resize}, msgs...)
}

func historyStory(lines []string, msgs ...tea.Msg) tea.Model {
	history := tui.NewHistoryWidget(tui.NewKeyMap())
	for _, line := range lines {
		history.AppendLine(line)
	}

	return widgetStory(history, func(widget *tui.HistoryWidget, size tea.WindowSizeMsg) {
		widget.SetHeight(size.Height)
	}, msgs...)
}

func appStory(msgs []tea.Msg) tea.Model {
	app := tui.NewUncorsApp(
		events.NewBus().Subscribe(1),
		make(chan events.Command, commandBuffer),
		version,
		tui.WithMemoryWidget(fixedMemoryWidget(memSmall)),
	)

	return replay(app, msgs...)
}

func fixedMemoryWidget(mb float64) *tui.MemoryWidget {
	return tui.NewMemoryWidget(tui.WithMemoryUsage(func() float64 { return mb }))
}

func key(code rune) tea.Msg {
	return tea.KeyPressMsg(tea.Key{Text: string(code), Code: code})
}

type staticModel string

func (m staticModel) Init() tea.Cmd {
	return nil
}

func (m staticModel) Update(tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m staticModel) View() tea.View {
	return tea.NewView(string(m))
}
