package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/evg4b/uncors/internal/config"
	"github.com/spf13/pflag"
)

// RunUncors loads the configuration from env.Args and runs the proxy until ctx
// is cancelled, a shutdown signal arrives, or the user quits. The --version and
// --help flags print their output and return without starting anything.
func RunUncors(ctx context.Context, env Env) error {
	uncorsConfig, cfgPath, err := config.LoadConfiguration(env.Fs, env.usage, env.Args)
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
