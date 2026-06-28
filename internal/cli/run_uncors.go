package cli

import (
	"context"
	"os"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/di"
	"github.com/spf13/afero"
)

func RunUncors(ctx context.Context, fs afero.Fs, args []string) error {
	uncorsConfig, path, err := config.LoadConfiguration(fs, args)
	if err != nil {
		return err
	}

	container := di.NewContainer(
		di.WithFs(fs),
		di.WithStdout(os.Stdout),
		di.WithVersion(Version),
	)
	defer container.Close()

	if uncorsConfig.Interactive {
		return runIneractive(ctx, container, uncorsConfig, path, args)
	}

	return runNonIneractive(ctx, container, uncorsConfig, path, args)
}
