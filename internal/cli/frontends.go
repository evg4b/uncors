package cli

import (
	tea "charm.land/bubbletea/v2"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/evg4b/uncors/internal/tui/render"
)

type frontend func(subscription *events.Subscription, commands chan<- events.Command) error

func headless(env Env) frontend {
	return func(subscription *events.Subscription, _ chan<- events.Command) error {
		output := render.NewCliOutput(env.Stdout)
		renderer := render.NewEventRenderer(output, env.Build.Version())

		for event := range subscription.Events() {
			renderer.Render(event)
		}

		if dropped := subscription.Dropped(); dropped > 0 {
			output.Warnf("%d events were dropped to keep the proxy responsive", dropped)
		}

		return nil
	}
}

func interactive(env Env) frontend {
	return func(subscription *events.Subscription, commands chan<- events.Command) error {
		app := tui.NewUncorsApp(subscription, commands, env.Build.Version())

		_, err := tea.NewProgram(app).Run()

		return err
	}
}
