package cli

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/di"
	uncor "github.com/evg4b/uncors/internal/uncors_app"
)

func runIneractive(
	ctx context.Context,
	container *di.Container,
	cfg *config.UncorsConfig,
	cfgPath string,
) error {
	app := uncor.NewUncorsApp(
		container,
		cfgPath,
		cfg,
		func() *config.UncorsConfig {
			reloaded, _, _ := config.LoadConfiguration(container.Fs(), container.Version(), container.Args())

			return reloaded
		},
	)

	_, err := tea.NewProgram(app, tea.WithContext(ctx)).
		Run()

	return err
}
