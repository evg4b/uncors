// Package stories shows each TUI widget, and the whole app, in one state with
// fixed data, like Storybook stories. tools/tuiplay shows them live, and
// TestStories snapshots their first frame at several widths.
package stories

import (
	"errors"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/tui"
)

const (
	screenHeight  = 24
	historyHeight = 10
	memSmall      = 42.5
	memLarge      = 2048.25
	overflowLines = 30
)

type Story struct {
	// Name is "<widget>/<state>".
	Name string
	// Height is the terminal height the story runs in; 0 fits the content.
	Height int
	New    func() tea.Model
}

var errReload = errors.New("mappings[0].from: invalid URL")

func Stories() []Story {
	return []Story{
		{Name: "styles/palette", New: func() tea.Model {
			return &paletteModel{}
		}},
		{Name: "help/short", New: func() tea.Model {
			return widgetStory(tui.NewHelpWidget(tui.NewKeyMap()), nil)
		}},
		{Name: "help/full", New: func() tea.Model {
			return widgetStory(tui.NewHelpWidget(tui.NewKeyMap()), nil, key('?'))
		}},
		{Name: "memory/small", New: func() tea.Model {
			return widgetStory(fixedMemoryWidget(memSmall), nil)
		}},
		{Name: "memory/large", New: func() tea.Model {
			return widgetStory(fixedMemoryWidget(memLarge), nil)
		}},
		{Name: "tracker/single", New: func() tea.Model {
			return widgetStory(tui.NewTrackerWidget(), nil, pending()[0])
		}},
		{Name: "tracker/many", New: func() tea.Model {
			return widgetStory(tui.NewTrackerWidget(), nil, pending()...)
		}},
		{Name: "tracker/long-url", New: func() tea.Model {
			return widgetStory(tui.NewTrackerWidget(), nil, started(longRequest)...)
		}},
		{Name: "tracker/handlers", New: func() tea.Model {
			return widgetStory(tui.NewTrackerWidget(), nil, started(handlerRequests()...)...)
		}},
		{Name: "history/short", Height: historyHeight, New: func() tea.Model {
			return historyStory(rendered(traffic()...))
		}},
		{Name: "history/handlers", Height: historyHeight, New: func() tea.Model {
			return historyStory(rendered(completed(handlerRequests()...)...))
		}},
		{Name: "history/statuses", Height: historyHeight, New: func() tea.Model {
			return historyStory(rendered(statuses()...))
		}},
		{Name: "history/overflow", Height: historyHeight, New: func() tea.Model {
			return historyStory(numberedLines(overflowLines))
		}},
		{Name: "history/scrolled-top", Height: historyHeight, New: func() tea.Model {
			return historyStory(numberedLines(overflowLines), key('g'))
		}},
		{Name: "app/started", Height: screenHeight, New: func() tea.Model {
			return appStory(serverStarted())
		}},
		{Name: "app/busy", Height: screenHeight, New: func() tea.Model {
			return appStory(append(serverStarted(), pending()...))
		}},
		{Name: "app/full-help", Height: screenHeight, New: func() tea.Model {
			return appStory(append(serverStarted(), key('?')))
		}},
		{Name: "app/notices", Height: screenHeight, New: func() tea.Model {
			return appStory(eventMsgs(
				events.ServerStarted{Mappings: mappings},
				events.Notice{Level: events.NoticeInfo, Text: "Recording HAR to ./traffic.har"},
				events.Notice{Level: events.NoticeWarning, Text: "Cache directory is not writable, caching in memory"},
				events.Notice{Level: events.NoticeError, Text: "script mocks/login.lua: attempt to index a nil value"},
				events.ReloadFailed{Err: errReload},
				events.ServerRestarted{Mappings: mappings},
				events.NewVersionAvailable{Current: "0.5.0", Latest: "0.6.0"},
			))
		}},
	}
}

func Find(name string) (Story, bool) {
	for _, story := range Stories() {
		if story.Name == name {
			return story, true
		}
	}

	return Story{}, false
}

// Render draws the story's first frame in a terminal of the given width the
// way Bubble Tea does: lines longer than the width are cut, not wrapped.
func Render(story Story, width int) string {
	height := story.Height
	model, _ := story.New().Update(tea.WindowSizeMsg{Width: width, Height: max(height, screenHeight)})
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
