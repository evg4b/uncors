package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/evg4b/uncors/internal/app"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/di"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/spf13/pflag"
)

// RunUncors loads the configuration from the container's args and runs the
// proxy until ctx is cancelled or a shutdown signal arrives. The --version and
// --help flags print their output and return without starting anything.
func RunUncors(ctx context.Context, container *di.Container) error {
	uncorsConfig, cfgPath, err := config.LoadConfiguration(
		container.Fs(),
		container.Version(),
		container.Args(),
		// Only the startup parse can reach --help, and drawing it is the CLI's
		// job, not the config package's.
		config.WithUsage(tui.PrintUsage),
	)
	if err != nil {
		switch {
		case errors.Is(err, config.ErrVersionRequested):
			_, err = fmt.Fprintln(container.Stdout(), container.Version())

			return err
		case errors.Is(err, pflag.ErrHelp):
			return nil
		default:
			return err
		}
	}

	if uncorsConfig.Interactive {
		return runInteractive(ctx, container, uncorsConfig, cfgPath)
	}

	return runNonInteractive(ctx, container, uncorsConfig, cfgPath)
}

// configLoader re-reads the configuration the process was started with. Both
// run modes reload the same way, so they share one loader.
func configLoader(container *di.Container) app.Loader {
	return func() (*config.UncorsConfig, error) {
		reloaded, _, err := config.LoadConfiguration(container.Fs(), container.Version(), container.Args())

		return reloaded, err
	}
}
