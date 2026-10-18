package tui_test

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/evg4b/uncors/internal/tui/styles"

	"charm.land/lipgloss/v2"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/tui"
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
			t.Run("should print single line", testutils.WithTrueColor(func(t *testing.T) {
				var buf strings.Builder
				tui.NewCliOutput(&buf).Request(events.RequestCompleted{Data: testCase.data})
				output := strings.Trim(buf.String(), "\n")
				assert.Equal(t, 1, lipgloss.Height(output))
			}))

			t.Run("should match snapshot", testutils.WithTrueColor(func(t *testing.T) {
				var buf strings.Builder
				tui.NewCliOutput(&buf).Request(events.RequestCompleted{Data: testCase.data})
				testutils.MatchSnapshot(t, buf.String())
			}))
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
		t.Run(testCase.name, testutils.WithTrueColor(func(t *testing.T) {
			testutils.MatchSnapshot(t, tui.FormatRequest(testCase.event))
		}))
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

		assert.Empty(t, styles.HandlerLabel(events.HandlerNone))
	})

	t.Run("hides the error of a cancelled request", func(t *testing.T) {
		cancelled := makeRequestData(http.MethodGet, "https://api.domain.com/api/user", http.StatusOK)
		cancelled.Cancelled = true

		line := tui.FormatRequest(events.RequestCompleted{Data: cancelled, Err: context.Canceled})

		assert.Equal(t, 1, lipgloss.Height(line))
		assert.NotContains(t, line, context.Canceled.Error())
	})
}
