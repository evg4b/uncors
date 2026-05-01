package tui_test

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

func makeRequestData(method, rawURL string, code int) *contracts.ReqestData {
	u, _ := url.Parse(rawURL)

	return &contracts.ReqestData{
		Method: method,
		URL:    u,
		Code:   code,
	}
}

func TestPrintResponse(t *testing.T) {
	tests := []struct {
		name string
		data *contracts.ReqestData
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
				tui.NewCliOutput(&buf).Request(testCase.data)
				output := strings.Trim(buf.String(), "\n")
				assert.Equal(t, 1, lipgloss.Height(output))
			}))

			t.Run("should match snapshot", testutils.WithTrueColor(func(t *testing.T) {
				var buf strings.Builder
				tui.NewCliOutput(&buf).Request(testCase.data)
				testutils.MatchSnapshot(t, buf.String())
			}))
		})
	}
}

func TestPrintResponse_PanicsForUnsupportedCodes(t *testing.T) {
	out := tui.NewCliOutput(io.Discard)

	t.Run("code below 100", func(t *testing.T) {
		assert.Panics(t, func() {
			out.Request(makeRequestData(http.MethodGet, "https://example.com/", 99))
		})
	})

	t.Run("code above 599", func(t *testing.T) {
		assert.Panics(t, func() {
			out.Request(makeRequestData(http.MethodGet, "https://example.com/", 600))
		})
	})
}
