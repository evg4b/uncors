package cli

import (
	"context"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/di"
	uncor "github.com/evg4b/uncors/internal/uncors_app"
	"github.com/spf13/afero"
)

func runIneractive(ctx context.Context, fs afero.Fs, cfg *config.UncorsConfig, cfgPath string, args []string) error {
	container := di.NewContainer(
		di.WithFs(fs),
		di.WithStdout(os.Stdout),
		di.WithVersion(Version),
	)
	defer container.Close()

	app := uncor.NewUncorsApp(
		container,
		cfgPath,
		cfg,
		func() *config.UncorsConfig {
			reloaded, _, _ := config.LoadConfiguration(container.Fs(), args)

			return reloaded
		},
	)

	_, err := tea.NewProgram(app, tea.WithContext(ctx)).
		Run()

	return err
}
