package stories

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/evg4b/uncors/internal/tui/render"
	"github.com/evg4b/uncors/internal/tui/styles"
)

const version = "0.0.0-story"

type mappingsList string

func (m mappingsList) String() string {
	return string(m)
}

const mappings = mappingsList(
	"http://localhost:3000 => https://api.example.com\n" +
		"    mock: [GET] /v1/health\n" +
		"    static: /assets => ./public\n" +
		"https://localhost:3443 => https://cdn.example.com",
)

var requestTime = time.Date(2026, time.January, 2, 15, 4, 5, 0, time.UTC)

func parseURL(rawURL string) *url.URL {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}

	return parsed
}

type request struct {
	method  string
	url     string
	handler events.HandlerKind
	code    int
	rewrite *events.Rewrite
}

const usersURL = "https://api.example.com/v1/users"

var pendingRequests = []request{
	{method: http.MethodGet, url: usersURL, handler: events.HandlerProxy},
	{method: http.MethodPost, url: "http://localhost:3000/api/login", handler: events.HandlerMock},
	{
		method: http.MethodPut, url: "http://localhost:3000/v1/users/42", handler: events.HandlerScript,
		rewrite: &events.Rewrite{From: "/v1/users/42", To: "/v2/users/42"},
	},
	{method: http.MethodGet, url: "https://api.example.com/v1/catalog?page=2", handler: events.HandlerCache},
}

var longRequest = request{
	method: http.MethodGet, handler: events.HandlerProxy,
	url: "https://api.example.com/v1/search?query=very+long+search+terms" +
		"&filters=category:books,price:10-50&sort=relevance",
}

var completedRequests = []request{
	{http.MethodGet, usersURL, events.HandlerProxy, http.StatusOK, nil},
	{http.MethodPost, "http://localhost:3000/api/login", events.HandlerMock, http.StatusCreated, nil},
	{http.MethodGet, "http://localhost:3000/assets/app.js", events.HandlerStatic, http.StatusOK, nil},
	{http.MethodGet, "https://api.example.com/v1/catalog", events.HandlerCache, http.StatusNotModified, nil},
	{http.MethodOptions, usersURL, events.HandlerOptions, http.StatusNoContent, nil},
	{http.MethodGet, "https://api.example.com/v1/missing", events.HandlerProxy, http.StatusNotFound, nil},
	{http.MethodDelete, "https://api.example.com/v1/users/42", events.HandlerScript, http.StatusInternalServerError, nil},
}

func started(requests ...request) []tea.Msg {
	msgs := make([]tea.Msg, 0, len(requests))

	var requestID uint64

	for _, req := range requests {
		requestID++

		msgs = append(msgs, tui.EventMsg(events.RequestStarted{
			ID: requestID, Method: req.method, URL: parseURL(req.url),
			Handler: req.handler, Rewrite: req.rewrite, At: requestTime,
		}))
	}

	return msgs
}

func pending() []tea.Msg {
	return started(pendingRequests...)
}

func traffic() []events.Event {
	evts := make([]events.Event, 0, len(completedRequests))

	for index, req := range completedRequests {
		evts = append(evts, events.RequestCompleted{
			ID:       uint64(len(pendingRequests) + index + 1),
			Handler:  req.handler,
			Rewrite:  req.rewrite,
			Data:     &contracts.RequestData{Method: req.method, URL: parseURL(req.url), Code: req.code},
			Duration: time.Duration(index+1) * time.Millisecond,
		})
	}

	return evts
}

func serverStarted() []tea.Msg {
	return eventMsgs(append([]events.Event{events.ServerStarted{Mappings: mappings}}, traffic()...)...)
}

func eventMsgs(evts ...events.Event) []tea.Msg {
	msgs := make([]tea.Msg, 0, len(evts))
	for _, event := range evts {
		msgs = append(msgs, tui.EventMsg(event))
	}

	return msgs
}

func rendered(evts ...events.Event) []string {
	var buf bytes.Buffer

	renderer := render.NewEventRenderer(render.NewCliOutput(&buf), version)
	lines := make([]string, 0, len(evts))

	for _, event := range evts {
		renderer.Render(event)

		lines = append(lines, strings.TrimRight(buf.String(), "\n"))
		buf.Reset()
	}

	return lines
}

func numberedLines(count int) []string {
	lines := make([]string, 0, count)
	for i := range count {
		lines = append(lines, fmt.Sprintf("line %02d", i+1))
	}

	return lines
}

var handlerKinds = []events.HandlerKind{
	events.HandlerProxy,
	events.HandlerCache,
	events.HandlerMock,
	events.HandlerScript,
	events.HandlerStatic,
	events.HandlerOptions,
}

var statusCodes = []int{
	http.StatusSwitchingProtocols,
	http.StatusOK,
	http.StatusNotModified,
	http.StatusNotFound,
	http.StatusBadGateway,
}

func handlerRequests() []request {
	requests := make([]request, 0, len(handlerKinds))
	for _, kind := range handlerKinds {
		if kind == events.HandlerOptions {
			requests = append(requests, request{http.MethodOptions, usersURL, kind, http.StatusNoContent, nil})

			continue
		}

		requests = append(requests, request{http.MethodGet, usersURL, kind, http.StatusOK, nil})
	}

	return requests
}

func completed(requests ...request) []events.Event {
	evts := make([]events.Event, 0, len(requests))

	for index, req := range requests {
		evts = append(evts, events.RequestCompleted{
			ID:       uint64(index + 1),
			Handler:  req.handler,
			Data:     &contracts.RequestData{Method: req.method, URL: parseURL(req.url), Code: req.code},
			Duration: time.Millisecond,
		})
	}

	return evts
}

func statuses() []events.Event {
	requests := make([]request, 0, len(statusCodes))
	for _, code := range statusCodes {
		requests = append(requests, request{http.MethodGet, usersURL, events.HandlerProxy, code, nil})
	}

	return append(completed(requests...), events.RequestCompleted{
		ID:      uint64(len(requests) + 1),
		Handler: events.HandlerProxy,
		Data:    &contracts.RequestData{Method: http.MethodGet, URL: parseURL(usersURL), Cancelled: true},
	})
}

const (
	nonstandardStatus = 999
	paletteMarginY    = 1
	paletteMarginX    = 2
)

var paletteHandlerRows = [][]events.HandlerKind{
	{events.HandlerProxy, events.HandlerCache, events.HandlerMock},
	{events.HandlerScript, events.HandlerStatic, events.HandlerOptions},
}

var paletteStatusRows = [][]int{
	{http.StatusSwitchingProtocols, http.StatusOK, http.StatusNotModified, http.StatusNotFound},
	{http.StatusBadGateway, nonstandardStatus},
}

var (
	lightTerminal = lipgloss.NewStyle().Background(lipgloss.Color("#FFFFFF")).Foreground(lipgloss.Color("#000000"))
	darkTerminal  = lipgloss.NewStyle().Background(lipgloss.Color("#1E1E1E")).Foreground(lipgloss.Color("#D4D4D4"))
)

func palette() string {
	return lipgloss.JoinHorizontal(lipgloss.Top,
		paletteColumn("light", styles.NewTheme(false), lightTerminal),
		" ",
		paletteColumn("dark", styles.NewTheme(true), darkTerminal),
	)
}

func paletteColumn(name string, theme styles.Theme, terminal lipgloss.Style) string {
	row := func(items ...string) string {
		return strings.Join(items, terminal.Render(" "))
	}
	text := func(style lipgloss.Style, value string) string {
		return style.Inherit(terminal).Render(value)
	}

	handlerLines := make([]string, 0, len(paletteHandlerRows))
	for _, kinds := range paletteHandlerRows {
		labels := make([]string, 0, len(kinds))
		for _, kind := range kinds {
			labels = append(labels, theme.HandlerLabel(kind))
		}

		handlerLines = append(handlerLines, row(labels...))
	}

	statusLines := make([]string, 0, len(paletteStatusRows))
	for _, codes := range paletteStatusRows {
		blocks := make([]string, 0, len(codes))
		for _, code := range codes {
			blocks = append(blocks, theme.StatusStyle(code).Render(strconv.Itoa(code)))
		}

		statusLines = append(statusLines, row(blocks...))
	}

	last := len(statusLines) - 1
	statusLines[last] = row(statusLines[last], theme.Cancelled.Render("CXL"), theme.PendingBlock.Render("▱▱▱"))

	levels := row(
		theme.InfoBlock.Render(render.InfoLabel),
		theme.WarningBlock.Render(render.WarningLabel),
		theme.ErrorBlock.Render(render.ErrorLabel),
	)
	texts := row(
		text(theme.URL, "url"),
		text(theme.DimText, "dim"),
		text(theme.MutedText, "muted"),
		text(theme.LogoRed, "red"),
		text(theme.LogoYellow, "yellow"),
	)

	lines := slices.Concat([]string{text(terminal.Bold(true), name)}, handlerLines, statusLines, []string{levels, texts})

	return terminal.Padding(paletteMarginY, paletteMarginX).Render(strings.Join(lines, "\n"))
}
