package stories

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/tui"
	uncorsapp "github.com/evg4b/uncors/internal/uncors_app"
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

		msgs = append(msgs, uncorsapp.EventMsg(events.RequestStarted{
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
			ID:       uint64(len(pendingRequests) + index + 1), //nolint:gosec // small positive index
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
		msgs = append(msgs, uncorsapp.EventMsg(event))
	}

	return msgs
}

func rendered(evts ...events.Event) []string {
	var buf bytes.Buffer

	renderer := tui.NewEventRenderer(tui.NewCliOutput(&buf), version)
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
