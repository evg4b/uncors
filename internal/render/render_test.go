package render_test

import (
	"bytes"
	"testing"

	"github.com/evg4b/uncors/internal/app"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/render"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errReload = assert.AnError

func mappings() config.Mappings {
	return config.Mappings{
		{From: hosts.Localhost.HTTPPort(3000), To: hosts.Github.HTTPS()},
	}
}

func renderOne(t *testing.T, event app.Event) string {
	t.Helper()

	var buf bytes.Buffer

	render.New(tui.NewCliOutput(&buf), "v1.2.3").Render(event)

	return buf.String()
}

// The rendered console output is the user-visible contract of this refactor.
// These snapshots are what prove moving it out of di.Proxy changed nothing.
func TestRenderLifecycle(t *testing.T) {
	testCases := []struct {
		name  string
		event app.Event
	}{
		{
			name:  "startup banner",
			event: app.LifecycleEvent{State: app.StateStarting, Mappings: mappings()},
		},
		{
			name:  "started is silent, the banner already said it",
			event: app.LifecycleEvent{State: app.StateStarted, Mappings: mappings()},
		},
		{
			name:  "reloading",
			event: app.LifecycleEvent{State: app.StateReloading},
		},
		{
			name:  "reloaded",
			event: app.LifecycleEvent{State: app.StateReloaded, Mappings: mappings()},
		},
		{
			name:  "reload failed",
			event: app.LifecycleEvent{State: app.StateReloadFailed, Err: errReload},
		},
		{
			name:  "start failed",
			event: app.LifecycleEvent{State: app.StateStartFailed, Err: errReload},
		},
		{
			name:  "interrupted stop moves past the echoed ^C",
			event: app.LifecycleEvent{State: app.StateStopping, Interrupted: true},
		},
		{
			name:  "non-interrupted stop prints nothing",
			event: app.LifecycleEvent{State: app.StateStopping},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testutils.MatchSnapshot(t, renderOne(t, testCase.event))
		})
	}
}

func TestRenderLog(t *testing.T) {
	levels := map[string]app.Level{
		"info":  app.LevelInfo,
		"warn":  app.LevelWarn,
		"error": app.LevelError,
	}

	for name, level := range levels {
		t.Run(name, func(t *testing.T) {
			testutils.MatchSnapshot(t, renderOne(t, app.LogEvent{Level: level, Message: "something happened"}))
		})
	}
}

func TestConsumeDrainsUntilTheStreamCloses(t *testing.T) {
	var buf bytes.Buffer

	events := make(chan app.Event, 2)

	for _, message := range []string{"first", "second"} {
		events <- app.LogEvent{Level: app.LevelInfo, Message: message}
	}

	close(events)

	render.New(tui.NewCliOutput(&buf), "v1.2.3").Consume(events)

	assert.Contains(t, buf.String(), "first")
	assert.Contains(t, buf.String(), "second")
}

func TestStoppedIsSilent(t *testing.T) {
	require.Empty(t, renderOne(t, app.LifecycleEvent{State: app.StateStopped}))
}
