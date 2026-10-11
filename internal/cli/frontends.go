package cli

import (
	tea "charm.land/bubbletea/v2"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/tui"
	uncorsapp "github.com/evg4b/uncors/internal/uncors_app"
)

// frontend presents the proxy to the user. It renders the events from
// subscription until it closes, and may send commands to the supervisor. It
// runs on its own goroutine; returning early is treated as a request to shut
// down.
type frontend func(subscription *events.Subscription, commands chan<- events.Command) error

// headless prints every event as plain console output.
func headless(env Env) frontend {
	return func(subscription *events.Subscription, _ chan<- events.Command) error {
		output := tui.NewCliOutput(env.Stdout)
		renderer := tui.NewEventRenderer(output, env.Version)

		for event := range subscription.Events() {
			renderer.Render(event)
		}

		if dropped := subscription.Dropped(); dropped > 0 {
			output.Warnf("%d events were dropped to keep the proxy responsive", dropped)
		}

		return nil
	}
}

// interactive runs the terminal UI.
func interactive(env Env) frontend {
	return func(subscription *events.Subscription, commands chan<- events.Command) error {
		app := uncorsapp.NewUncorsApp(subscription, commands, env.Version)

		_, err := tea.NewProgram(app).Run()

		return err
	}
}
