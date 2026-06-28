package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/di"
	"github.com/spf13/pflag"
)

func RunUncors(ctx context.Context, container *di.Container) error {
	uncorsConfig, path, err := config.LoadConfiguration(container.Fs(), container.Version(), container.Args())
	if err != nil {
		if errors.Is(err, config.ErrVersionRequested) {
			fmt.Fprintln(os.Stdout, container.Version())

			return nil
		}

		if errors.Is(err, pflag.ErrHelp) {
			return nil
		}

		return err
	}

	var runError error
	if uncorsConfig.Interactive {
		runError = runIneractive(ctx, container, uncorsConfig, path)
	} else {
		runError = runNonIneractive(ctx, container, uncorsConfig, path)
	}

	if runError != nil && !errors.Is(runError, pflag.ErrHelp) {
		return runError
	}

	return nil
}
