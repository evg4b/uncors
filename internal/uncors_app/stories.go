package uncorsapp

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/tui"
)

// Stories show each widget alone with fixed data. tools/tuiplay runs them
// live, and TestStories snapshots their first frame at several widths.

const (
	storyVersion       = "0.0.0-story"
	storyScreenHeight  = 24
	storyHistoryHeight = 10
	storyMemSmall      = 42.5
	storyMemLarge      = 2048.25
	storyOverflowLines = 30
	storyCommandBuffer = 16
)

type Story struct {
	// Name is "<widget>/<state>".
	Name string
	// Height is the terminal height the story runs in; 0 fits the content.
	Height int
	New    func() tea.Model
}

func Stories() []Story {
	return []Story{
		{Name: "help/short", New: func() tea.Model {
			return widgetStory(NewHelpWidget(newKeyMap()), nil)
		}},
		{Name: "help/full", New: func() tea.Model {
			return widgetStory(NewHelpWidget(newKeyMap()), nil, storyKey('?'))
		}},
		{Name: "memory/small", New: func() tea.Model {
			return widgetStory(fixedMemoryWidget(storyMemSmall), nil)
		}},
		{Name: "memory/large", New: func() tea.Model {
			return widgetStory(fixedMemoryWidget(storyMemLarge), nil)
		}},
		{Name: "tracker/single", New: func() tea.Model {
			return widgetStory(NewTrackerWidget(), nil, storyPending()[0])
		}},
		{Name: "tracker/many", New: func() tea.Model {
			return widgetStory(NewTrackerWidget(), nil, storyPending()...)
		}},
		{Name: "tracker/long-url", New: func() tea.Model {
			return widgetStory(NewTrackerWidget(), nil, storyStartedMsgs(storyLongRequest)...)
		}},
		{Name: "history/short", Height: storyHistoryHeight, New: func() tea.Model {
			return historyStory(storyRendered(storyTraffic()...))
		}},
		{Name: "history/overflow", Height: storyHistoryHeight, New: func() tea.Model {
			return historyStory(storyNumberedLines(storyOverflowLines))
		}},
		{Name: "history/scrolled-top", Height: storyHistoryHeight, New: func() tea.Model {
			return historyStory(append(storyNumberedLines(storyOverflowLines), storyKey('g')))
		}},
		{Name: "app/started", Height: storyScreenHeight, New: func() tea.Model {
			return appStory(storyStarted())
		}},
		{Name: "app/busy", Height: storyScreenHeight, New: func() tea.Model {
			return appStory(append(storyStarted(), storyPending()...))
		}},
		{Name: "app/full-help", Height: storyScreenHeight, New: func() tea.Model {
			return appStory(append(storyStarted(), storyKey('?')))
		}},
		{Name: "app/notices", Height: storyScreenHeight, New: func() tea.Model {
			return appStory(storyLifecycle(
				events.ServerStarted{Mappings: storyMappings},
				events.Notice{Level: events.NoticeInfo, Text: "Recording HAR to ./traffic.har"},
				events.Notice{Level: events.NoticeWarning, Text: "Cache directory is not writable, caching in memory"},
				events.Notice{Level: events.NoticeError, Text: "script mocks/login.lua: attempt to index a nil value"},
				events.ReloadFailed{Err: errStoryReload},
				events.ServerRestarted{Mappings: storyMappings},
				events.NewVersionAvailable{Current: "0.5.0", Latest: "0.6.0"},
			))
		}},
	}
}

// RenderStory draws the story's first frame in a terminal of the given width
// the way Bubble Tea does: lines longer than the width are cut, not wrapped.
func RenderStory(story Story, width int) string {
	height := story.Height
	model, _ := story.New().Update(tea.WindowSizeMsg{Width: width, Height: max(height, storyScreenHeight)})
	content := model.View().Content

	if height == 0 {
		height = lipgloss.Height(content)
	}

	return uv.TrimSpace(DrawFrame(content, width, height))
}

// DrawFrame clips content to a width x height screen and returns it with its
// styles, every line padded to the width; ansi.Strip gives the plain text.
func DrawFrame(content string, width, height int) string {
	screen := uv.NewScreenBuffer(width, height)
	uv.NewStyledString(content).Draw(screen, screen.Bounds())

	return screen.Render()
}

type storyWidget[T any] interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (T, tea.Cmd)
	View() tea.View
}

// widgetModel adapts a widget, whose Update returns its own type, to tea.Model.
type widgetModel[T storyWidget[T]] struct {
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

func widgetStory[T storyWidget[T]](widget T, resize func(T, tea.WindowSizeMsg), msgs ...tea.Msg) tea.Model {
	return replay(&widgetModel[T]{widget: widget, resize: resize}, msgs...)
}

func historyStory(msgs []tea.Msg) tea.Model {
	return widgetStory(NewHistoryWidget(newKeyMap()), func(widget *HistoryWidget, size tea.WindowSizeMsg) {
		widget.SetHeight(size.Height)
	}, msgs...)
}

func appStory(msgs []tea.Msg) tea.Model {
	app := NewUncorsApp(
		events.NewBus().Subscribe(1),
		make(chan events.Command, storyCommandBuffer),
		storyVersion,
	)
	app.memWidget = fixedMemoryWidget(storyMemSmall)

	return replay(app, msgs...)
}

func fixedMemoryWidget(mb float64) *MemoryWidget {
	return &MemoryWidget{memMB: mb, getMemUsage: func() float64 { return mb }}
}

func storyKey(code rune) tea.Msg {
	return tea.KeyPressMsg(tea.Key{Text: string(code), Code: code})
}

type storyMappingsList string

func (m storyMappingsList) String() string {
	return string(m)
}

const storyMappings = storyMappingsList(
	"http://localhost:3000 => https://api.example.com\n" +
		"    mock: [GET] /v1/health\n" +
		"    static: /assets => ./public\n" +
		"https://localhost:3443 => https://cdn.example.com",
)

var errStoryReload = errors.New("mappings[0].from: invalid URL")

var storyTime = time.Date(2026, time.January, 2, 15, 4, 5, 0, time.UTC)

func storyURL(rawURL string) *url.URL {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}

	return parsed
}

type storyRequest struct {
	method  string
	url     string
	handler events.HandlerKind
	code    int
	rewrite *events.Rewrite
}

const storyUsersURL = "https://api.example.com/v1/users"

var storyPendingRequests = []storyRequest{
	{method: http.MethodGet, url: storyUsersURL, handler: events.HandlerProxy},
	{method: http.MethodPost, url: "http://localhost:3000/api/login", handler: events.HandlerMock},
	{
		method: http.MethodPut, url: "http://localhost:3000/v1/users/42", handler: events.HandlerScript,
		rewrite: &events.Rewrite{From: "/v1/users/42", To: "/v2/users/42"},
	},
	{method: http.MethodGet, url: "https://api.example.com/v1/catalog?page=2", handler: events.HandlerCache},
}

var storyLongRequest = storyRequest{
	method: http.MethodGet, handler: events.HandlerProxy,
	url: "https://api.example.com/v1/search?query=very+long+search+terms" +
		"&filters=category:books,price:10-50&sort=relevance",
}

var storyCompletedRequests = []storyRequest{
	{http.MethodGet, storyUsersURL, events.HandlerProxy, http.StatusOK, nil},
	{http.MethodPost, "http://localhost:3000/api/login", events.HandlerMock, http.StatusCreated, nil},
	{http.MethodGet, "http://localhost:3000/assets/app.js", events.HandlerStatic, http.StatusOK, nil},
	{http.MethodGet, "https://api.example.com/v1/catalog", events.HandlerCache, http.StatusNotModified, nil},
	{http.MethodOptions, storyUsersURL, events.HandlerOptions, http.StatusNoContent, nil},
	{http.MethodGet, "https://api.example.com/v1/missing", events.HandlerProxy, http.StatusNotFound, nil},
	{http.MethodDelete, "https://api.example.com/v1/users/42", events.HandlerScript, http.StatusInternalServerError, nil},
}

func storyStartedMsgs(requests ...storyRequest) []tea.Msg {
	msgs := make([]tea.Msg, 0, len(requests))

	var requestID uint64

	for _, req := range requests {
		requestID++

		msgs = append(msgs, requestStartedMsg{
			ID: requestID, Method: req.method, URL: storyURL(req.url),
			Handler: req.handler, Rewrite: req.rewrite, At: storyTime,
		})
	}

	return msgs
}

func storyPending() []tea.Msg {
	return storyStartedMsgs(storyPendingRequests...)
}

func storyTraffic() []events.Event {
	evts := make([]events.Event, 0, len(storyCompletedRequests))

	for index, req := range storyCompletedRequests {
		evts = append(evts, events.RequestCompleted{
			ID:       uint64(len(storyPendingRequests) + index + 1), //nolint:gosec // small positive index
			Handler:  req.handler,
			Rewrite:  req.rewrite,
			Data:     &contracts.RequestData{Method: req.method, URL: storyURL(req.url), Code: req.code},
			Duration: time.Duration(index+1) * time.Millisecond,
		})
	}

	return evts
}

func storyStarted() []tea.Msg {
	return storyLifecycle(append([]events.Event{events.ServerStarted{Mappings: storyMappings}}, storyTraffic()...)...)
}

// storyLifecycle wraps events the way watchEventsCmd delivers them to the app.
func storyLifecycle(evts ...events.Event) []tea.Msg {
	msgs := make([]tea.Msg, 0, len(evts))

	for _, event := range evts {
		if completed, ok := event.(events.RequestCompleted); ok {
			msgs = append(msgs, requestCompletedMsg(completed))

			continue
		}

		msgs = append(msgs, lifecycleMsg{event: event})
	}

	return msgs
}

// storyRendered renders events the way the app feeds them to the history.
func storyRendered(evts ...events.Event) []tea.Msg {
	var buf bytes.Buffer

	renderer := tui.NewEventRenderer(tui.NewCliOutput(&buf), storyVersion)
	msgs := make([]tea.Msg, 0, len(evts))

	for _, event := range evts {
		renderer.Render(event)

		msgs = append(msgs, outputLineMsg(strings.TrimRight(buf.String(), "\n")))
		buf.Reset()
	}

	return msgs
}

func storyNumberedLines(count int) []tea.Msg {
	msgs := make([]tea.Msg, 0, count)

	for i := range count {
		msgs = append(msgs, outputLineMsg(fmt.Sprintf("line %02d", i+1)))
	}

	return msgs
}
