package render_test

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/tui/render"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/stretchr/testify/assert"
)

var errReload = errors.New("mappings must not be empty")

func renderEvent(event events.Event) string {
	var output strings.Builder

	render.NewEventRenderer(render.NewCliOutput(&output), "1.2.3").Render(event)

	return ansi.Strip(output.String())
}

func TestEventRenderer(t *testing.T) {
	mappings := config.Mappings{{From: hosts.Localhost.HTTPPort(3000), To: hosts.Github.HTTPS()}}

	tests := []struct {
		name     string
		event    events.Event
		contains []string
	}{
		{
			name:     "server started",
			event:    events.ServerStarted{Mappings: mappings},
			contains: []string{"version: 1.2.3", "DON'T USE IT FOR PRODUCTION!", "http://localhost:3000 => https://github.com"},
		},
		{
			name:     "server restarted",
			event:    events.ServerRestarted{Mappings: mappings},
			contains: []string{"Server restarted", "http://localhost:3000 => https://github.com"},
		},
		{
			name:     "server stopped",
			event:    events.ServerStopped{},
			contains: []string{"Server was stopped"},
		},
		{
			name:     "reload failed",
			event:    events.ReloadFailed{Err: errReload},
			contains: []string{"Failed to reload configuration, keeping the previous one: mappings must not be empty"},
		},
		{
			name:     "new version",
			event:    events.NewVersionAvailable{Current: "0.6.0", Latest: "0.7.0"},
			contains: []string{"NEW VERSION IS AVAILABLE!", "0.6.0 is not the latest version, you should upgrade to 0.7.0"},
		},
		{
			name:     "info notice",
			event:    events.Notice{Level: events.NoticeInfo, Text: "informing"},
			contains: []string{"INFO", "informing"},
		},
		{
			name:     "warning notice",
			event:    events.Notice{Level: events.NoticeWarning, Text: "warning"},
			contains: []string{"WARN", "warning"},
		},
		{
			name:     "error notice",
			event:    events.Notice{Level: events.NoticeError, Text: "failing"},
			contains: []string{"ERROR", "failing"},
		},
		{
			name: "completed request",
			event: events.RequestCompleted{
				Handler: events.HandlerMock,
				Data:    makeRequestData(http.MethodGet, "http://localhost:3000/api", http.StatusOK),
			},
			contains: []string{"MOCK", "200 GET", "http://localhost:3000/api"},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			output := renderEvent(testCase.event)

			for _, expected := range testCase.contains {
				assert.Contains(t, output, expected)
			}
		})
	}

	t.Run("started requests print nothing", func(t *testing.T) {
		assert.Empty(t, renderEvent(events.RequestStarted{ID: 1}))
	})
}
