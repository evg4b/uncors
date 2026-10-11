package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/evg4b/uncors/internal/config"
	"github.com/spf13/pflag"
)

func RunUncors(ctx context.Context, env Env) error {
	uncorsConfig, cfgPath, err := config.LoadConfiguration(ctx, env.Fs, env.usage, env.Args)
	if err != nil {
		switch {
		case errors.Is(err, config.ErrVersionRequested):
			_, err = fmt.Fprintln(env.Stdout, env.Version)

			return err
		case errors.Is(err, pflag.ErrHelp):
			return nil
		default:
			return err
		}
	}

	frontend := headless(env)
	if uncorsConfig.Interactive {
		frontend = interactive(env)
	}

	return supervise(ctx, env, uncorsConfig, cfgPath, frontend)
}
