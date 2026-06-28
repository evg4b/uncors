package cli

import (
	"context"
	"errors"
	"os"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/di"
	"github.com/spf13/afero"
	"github.com/spf13/pflag"
)

func RunUncors(ctx context.Context, fs afero.Fs, args []string) error {
	var containerError error

	container := di.NewContainer(
		di.WithFs(fs),
		di.WithStdout(os.Stdout),
		di.WithVersion(Version),
	)
	defer func() {
		containerError = container.Close()
	}()

	uncorsConfig, path, err := config.LoadConfiguration(fs, Version, args)
	if err != nil {
		if errors.Is(err, config.ErrVersionRequested) {
			println(container.Version())
			return nil
		}

		if !errors.Is(err, pflag.ErrHelp) && !errors.Is(err, config.ErrVersionRequested) {
			return err
		}

		return nil
	}

	var runError error
	if uncorsConfig.Interactive {
		runError = runIneractive(ctx, container, uncorsConfig, path, args)
	} else {
		runError = runNonIneractive(ctx, container, uncorsConfig, path, args)
	}

	if runError != nil && !errors.Is(runError, pflag.ErrHelp) {
		return runError
	}

	return containerError
}
