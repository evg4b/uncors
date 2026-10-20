package tui_test

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/evg4b/uncors/internal/tui/styles"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

func makeRequestData(method, rawURL string, code int) *contracts.RequestData {
	u, _ := url.Parse(rawURL)

	return &contracts.RequestData{
		Method: method,
		URL:    u,
		Code:   code,
	}
}

func TestPrintResponse(t *testing.T) {
	tests := []struct {
		name string
		data *contracts.RequestData
	}{
		{
			name: "1xx informational",
			data: makeRequestData(http.MethodPost, "https://api.domain.com/api/info", 100),
		},
		{
			name: "2xx success",
			data: makeRequestData(http.MethodGet, "https://api.domain.com/help", 200),
		},
		{
			name: "3xx redirect",
			data: makeRequestData(http.MethodPatch, "https://api.domain.com/api/user", 301),
		},
		{
			name: "4xx client error",
			data: makeRequestData(http.MethodDelete, "https://api.domain.com/api/user/permission", 404),
		},
		{
			name: "5xx server error",
			data: makeRequestData(http.MethodPost, "https://api.domain.com/", 500),
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Run("should print single line", func(t *testing.T) {
				var buf strings.Builder
				tui.NewCliOutput(&buf).Request(events.RequestCompleted{Data: testCase.data})
				output := strings.Trim(buf.String(), "\n")
				assert.Equal(t, 1, lipgloss.Height(output))
			})

			t.Run("should match snapshot", func(t *testing.T) {
				var buf strings.Builder
				tui.NewCliOutput(&buf).Request(events.RequestCompleted{Data: testCase.data})
				testutils.MatchSnapshot(t, buf.String())
			})
		})
	}
}

func TestPrintResponse_NonstandardCodes(t *testing.T) {
	for _, code := range []int{99, 600, 799, 999} {
		t.Run(strconv.Itoa(code), func(t *testing.T) {
			var buf strings.Builder

			assert.NotPanics(t, func() {
				tui.NewCliOutput(&buf).Request(events.RequestCompleted{
					Data: makeRequestData(http.MethodGet, "https://example.com/", code),
				})
			})
			assert.Contains(t, buf.String(), strconv.Itoa(code)+" GET")
			assert.Equal(t, 1, lipgloss.Height(strings.Trim(buf.String(), "\n")))
		})
	}
}

func TestMethodLabel(t *testing.T) {
	tests := []struct {
		method string
		label  string
	}{
		{http.MethodGet, "GET"},
		{http.MethodPatch, "PATCH"},
		{http.MethodDelete, "DEL"},
		{http.MethodOptions, "OPTS"},
		{http.MethodConnect, "CONN"},
		{"PROPFIND", "PROP…"},
	}

	for _, testCase := range tests {
		t.Run(testCase.method, func(t *testing.T) {
			line := tui.FormatRequest(events.RequestCompleted{
				Data: makeRequestData(testCase.method, "https://example.com/", http.StatusOK),
			})

			assert.Contains(t, ansi.Strip(line), "200 "+testCase.label+" ")
		})
	}
}

func TestPrintResponse_LongMethods(t *testing.T) {
	methods := []string{http.MethodPatch, http.MethodTrace, http.MethodOptions, http.MethodConnect, "PROPFIND"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			var buf strings.Builder

			tui.NewCliOutput(&buf).Request(events.RequestCompleted{
				Data: makeRequestData(method, "https://example.com/", http.StatusBadGateway),
			})

			cancelled := makeRequestData(method, "https://example.com/", http.StatusOK)
			cancelled.Cancelled = true

			pending := events.RequestStarted{Method: method, URL: cancelled.URL, Handler: events.HandlerOptions}

			assert.Equal(t, 1, lipgloss.Height(strings.Trim(buf.String(), "\n")))
			assert.Equal(t, 1, lipgloss.Height(tui.FormatRequest(events.RequestCompleted{Data: cancelled})))
			assert.Equal(t, 1, lipgloss.Height(tui.FormatPending(pending, "▱▱▱")))
		})
	}
}

func TestFormatPending(t *testing.T) {
	reqURL, _ := url.Parse("http://localhost/test")

	line := ansi.Strip(tui.FormatPending(events.RequestStarted{
		Method:  http.MethodDelete,
		URL:     reqURL,
		Handler: events.HandlerScript,
		Rewrite: &events.Rewrite{From: "/test", To: "/v2/test"},
	}, "▱▱▱"))

	assert.Equal(t, " SCRPT  ▱▱▱ DEL    http://localhost/test → /v2/test", line)
}

var errUpstream = errors.New("dial tcp 127.0.0.1:1: connect: connection refused")

func TestFormatRequest(t *testing.T) {
	data := makeRequestData(http.MethodGet, "https://api.domain.com/api/user", http.StatusOK)

	tests := []struct {
		name  string
		event events.RequestCompleted
	}{
		{
			name:  "without handler",
			event: events.RequestCompleted{Data: data},
		},
		{
			name:  "proxy handler",
			event: events.RequestCompleted{Handler: events.HandlerProxy, Data: data},
		},
		{
			name:  "cache handler",
			event: events.RequestCompleted{Handler: events.HandlerCache, Data: data},
		},
		{
			name: "rewritten path",
			event: events.RequestCompleted{
				Handler: events.HandlerProxy,
				Rewrite: &events.Rewrite{From: "/api/user", To: "/api/v2/user"},
				Data:    data,
			},
		},
		{
			name: "handler error",
			event: events.RequestCompleted{
				Handler: events.HandlerProxy,
				Data:    makeRequestData(http.MethodGet, "https://api.domain.com/api/user", http.StatusInternalServerError),
				Err:     errUpstream,
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testutils.MatchSnapshot(t, tui.FormatRequest(testCase.event))
		})
	}

	t.Run("every handler label fits on one line", func(t *testing.T) {
		kinds := []events.HandlerKind{
			events.HandlerProxy, events.HandlerCache, events.HandlerMock,
			events.HandlerScript, events.HandlerStatic, events.HandlerOptions,
		}

		for _, kind := range kinds {
			label := styles.HandlerLabel(kind)
			assert.Equal(t, 1, lipgloss.Height(label), kind.String())
			assert.Contains(t, label, kind.String())
		}

		blank := styles.HandlerLabel(events.HandlerNone)
		assert.Empty(t, strings.TrimSpace(ansi.Strip(blank)))
		assert.Equal(t, lipgloss.Width(styles.HandlerLabel(events.HandlerProxy)), lipgloss.Width(blank))
	})

	t.Run("hides the error of a cancelled request", func(t *testing.T) {
		cancelled := makeRequestData(http.MethodGet, "https://api.domain.com/api/user", http.StatusOK)
		cancelled.Cancelled = true

		line := tui.FormatRequest(events.RequestCompleted{Data: cancelled, Err: context.Canceled})

		assert.Equal(t, 1, lipgloss.Height(line))
		assert.NotContains(t, line, context.Canceled.Error())
	})
}
