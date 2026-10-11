package cli

import (
	"errors"

	"github.com/evg4b/uncors/internal/commands"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/spf13/pflag"
)

const GenerateCertsCmd = "generate-certs"

// GenerateCerts runs the generate-certs command with env.Args as its flags.
func GenerateCerts(env Env) error {
	cmd := commands.NewGenerateCertsCommand(
		commands.WithOutput(tui.NewCliOutput(env.Stdout)),
		commands.WithFs(env.Fs),
	)

	flags := pflag.NewFlagSet(GenerateCertsCmd, pflag.ContinueOnError)
	cmd.DefineFlags(flags, env.Version)

	err := flags.Parse(env.Args)
	if err != nil {
		if !errors.Is(err, pflag.ErrHelp) {
			return err
		}

		return nil
	}

	return cmd.Execute()
}
