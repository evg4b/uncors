package cli

import (
	"context"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/di"
	uncor "github.com/evg4b/uncors/internal/uncors_app"
	"github.com/spf13/afero"
	"github.com/spf13/pflag"
)

const GenerateCertsCmd = "generate-certs"

func GenerateCerts(args []string) error {
	fs := afero.NewOsFs()

	container := di.NewContainer(
		di.WithFs(fs),
		di.WithStdout(os.Stdout),
		// di.WithVersion("Version"),
	)
	defer container.Close()

	cmd := container.GenerateCertsCommand()

	flags := pflag.NewFlagSet(GenerateCertsCmd, pflag.ContinueOnError)
	cmd.DefineFlags(flags)

	err := flags.Parse(args)
	if err != nil {
		return err
	}

	return cmd.Execute()
}

func RunUncors(ctx context.Context, fs afero.Fs, args []string) error {
	uncorsConfig, path, err := config.LoadConfiguration(fs, args)
	if err != nil {
		return err
	}

	if uncorsConfig.Interactive {
		return runIneractive(fs, uncorsConfig, path, args)
	}

	return runNonIneractive(ctx, fs, uncorsConfig, path, args)
}

func runIneractive(fs afero.Fs, uncorsConfig *config.UncorsConfig, configPath string, args []string) error {
	container := di.NewContainer(
		di.WithFs(fs),
		di.WithStdout(os.Stdout),
		// di.WithVersion("Version"),
	)
	defer container.Close()

	app := uncor.NewUncorsApp(
		container,
		configPath,
		uncorsConfig,
		func() *config.UncorsConfig {
			reloaded, _, _ := config.LoadConfiguration(container.Fs(), args)

			return reloaded
		},
	)

	_, err := tea.NewProgram(app).Run()

	return err
}
