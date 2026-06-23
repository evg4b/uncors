package cli

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/di"
	"github.com/evg4b/uncors/internal/tui"
	uncor "github.com/evg4b/uncors/internal/uncors_app"
	"github.com/spf13/afero"
	"github.com/spf13/pflag"
)

const (
	GenerateCertsCmd = "generate-certs"
	baseAddress      = "127.0.0.1"
)

func GenerateCerts(args []string) error {
	print("GenerateCerts ")

	pflag.Usage = func() {
		output := tui.NewCliOutput(os.Stdout)
		tui.PrintLogo(output, "Version")
		fmt.Fprintf(output, "Usage of %s:\n", os.Args[0])
		pflag.PrintDefaults()
	}

	flags := pflag.NewFlagSet(GenerateCertsCmd, pflag.ContinueOnError)

	flags.Parse(args)

	return nil
}

func RunUncors(args []string) error {
	fs := afero.NewOsFs()

	uncorsConfig, path, err := config.LoadConfiguration(fs, args)
	if err != nil {
		return err
	}

	if uncorsConfig.Interactive {
		return runIneractive(fs, uncorsConfig, path, args)
	}

	return runNonIneractive(fs, uncorsConfig, path, args)
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
