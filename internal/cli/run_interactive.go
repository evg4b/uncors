package cli

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/di"
	uncorsapp "github.com/evg4b/uncors/internal/uncors_app"
)

// runInteractive starts the proxy in interactive TUI mode. It drives the same
// app.Service headless mode does, including the same reload behaviour; the only
// difference is that here the events are rendered by the Bubble Tea model.
func runInteractive(
	ctx context.Context,
	container *di.Container,
	cfg *config.UncorsConfig,
	cfgPath string,
) error {
	app := uncorsapp.NewUncorsApp(container, cfgPath, cfg, configLoader(container))

	_, err := tea.NewProgram(app, tea.WithContext(ctx)).Run()

	return err
}
