package cli

import (
	"context"

	"github.com/evg4b/uncors/internal/config"
	"github.com/spf13/afero"
)

func RunUncors(ctx context.Context, fs afero.Fs, args []string) error {
	uncorsConfig, path, err := config.LoadConfiguration(fs, args)
	if err != nil {
		return err
	}

	if uncorsConfig.Interactive {
		return runIneractive(ctx, fs, uncorsConfig, path, args)
	}

	return runNonIneractive(ctx, fs, uncorsConfig, path, args)
}
