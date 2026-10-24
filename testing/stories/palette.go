package stories

import (
	"bytes"
	"errors"
	"image/color"
	"net/http"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/tui/render"
	"github.com/evg4b/uncors/internal/tui/styles"
)

const (
	paletteMarginX = 2
	paletteMarginY = 1
	paletteHost    = "http://api.local"
	pendingFrame   = "▱▱▱"
)

var errScript = errors.New("attempt to index a nil value")

type paletteTerminal struct {
	name  string
	theme *styles.Theme
	fg    color.Color
	bg    color.Color
}

var paletteTerminals = []paletteTerminal{
	{"light", styles.NewTheme(styles.Light), lipgloss.Color("#1F1F1F"), lipgloss.Color("#FFFFFF")},
	{"dark", styles.NewTheme(styles.Dark), lipgloss.Color("#D4D4D4"), lipgloss.Color("#1E1E1E")},
}

func completedSample(handler events.HandlerKind, method, path string, code int) events.RequestCompleted {
	return events.RequestCompleted{
		Handler: handler,
		Data:    &contracts.RequestData{Method: method, URL: parseURL(paletteHost + path), Code: code},
	}
}

func paletteEvents() []events.Event {
	failed := completedSample(events.HandlerScript, http.MethodDelete, "/users/42", http.StatusInternalServerError)
	failed.Err = errScript

	rewritten := completedSample(events.HandlerProxy, http.MethodGet, "/old", http.StatusFound)
	rewritten.Rewrite = &events.Rewrite{From: "/old", To: "/new"}

	cancelled := completedSample(events.HandlerProxy, http.MethodGet, "/slow", 0)
	cancelled.Data.Cancelled = true

	return []events.Event{
		completedSample(events.HandlerProxy, http.MethodGet, "/users", http.StatusOK),
		completedSample(events.HandlerCache, http.MethodGet, "/catalog", http.StatusOK),
		completedSample(events.HandlerMock, http.MethodPost, "/login", http.StatusCreated),
		failed,
		completedSample(events.HandlerStatic, http.MethodGet, "/app.js", http.StatusOK),
		completedSample(events.HandlerOptions, http.MethodOptions, "/users", http.StatusNoContent),
		completedSample(events.HandlerProxy, http.MethodGet, "/ws", http.StatusSwitchingProtocols),
		rewritten,
		completedSample(events.HandlerProxy, http.MethodGet, "/missing", http.StatusNotFound),
		cancelled,
		events.Notice{Level: events.NoticeInfo, Text: "Recording HAR to ./traffic.har"},
		events.Notice{Level: events.NoticeWarning, Text: "Cache directory is read-only"},
		events.Notice{Level: events.NoticeError, Text: "mocks/login.lua failed"},
	}
}

var paletteRequests = []request{
	{method: http.MethodGet, url: paletteHost + "/users", handler: events.HandlerProxy},
	{method: http.MethodPut, url: paletteHost + "/users/42", handler: events.HandlerScript},
}

func paletteSample(theme *styles.Theme) string {
	var buf bytes.Buffer

	renderer := render.NewEventRenderer(render.NewCliOutput(&buf, render.WithTheme(theme)), version)
	for _, event := range paletteEvents() {
		renderer.Render(event)
	}

	lines := []string{strings.TrimRight(buf.String(), "\n"), ""}

	formatter := render.NewFormatter(theme)
	for _, req := range paletteRequests {
		lines = append(lines, formatter.Pending(events.RequestStarted{
			Method: req.method, URL: parseURL(req.url), Handler: req.handler,
		}, pendingFrame))
	}

	return strings.Join(append(lines, "", theme.MutedText.Render("[ 42.5 MB ]")), "\n")
}

func onTerminal(terminal paletteTerminal, content string) string {
	content = lipgloss.NewStyle().Bold(true).Render(terminal.name) + "\n\n" + content

	width := lipgloss.Width(content) + paletteMarginX + paletteMarginX
	height := lipgloss.Height(content) + paletteMarginY + paletteMarginY

	screen := uv.NewScreenBuffer(width, height)
	uv.NewStyledString(content).Draw(screen, uv.Rect(paletteMarginX, paletteMarginY, width, height))

	for line := range height {
		for column := range width {
			cell := screen.CellAt(column, line)
			if cell == nil {
				continue
			}

			if cell.Style.Fg == nil {
				cell.Style.Fg = terminal.fg
			}

			if cell.Style.Bg == nil {
				cell.Style.Bg = terminal.bg
			}

			screen.SetCell(column, line, cell)
		}
	}

	return screen.Render()
}

type paletteModel struct {
	width int
}

func (m *paletteModel) Init() tea.Cmd {
	return nil
}

func (m *paletteModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = size.Width
	}

	return m, nil
}

func (m *paletteModel) View() tea.View {
	panels := make([]string, 0, len(paletteTerminals))
	spaced := make([]string, 0, len(paletteTerminals))

	for _, terminal := range paletteTerminals {
		panel := onTerminal(terminal, paletteSample(terminal.theme))
		panels = append(panels, panel)
		spaced = append(spaced, lipgloss.NewStyle().MarginRight(1).Render(panel))
	}

	side := lipgloss.JoinHorizontal(lipgloss.Top, spaced...)
	if lipgloss.Width(side) > m.width {
		return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, panels...))
	}

	return tea.NewView(side)
}
