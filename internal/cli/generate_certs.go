package cli

import (
	"context"
	"errors"

	"github.com/evg4b/uncors/internal/commands"
	"github.com/evg4b/uncors/internal/telemetry"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/spf13/pflag"
)

const GenerateCertsCmd = "generate-certs"

func GenerateCerts(env Env) error {
	cmd := commands.NewGenerateCertsCommand(
		commands.WithOutput(tui.NewCliOutput(env.Stdout)),
		commands.WithFs(env.Fs),
	)

	flags := pflag.NewFlagSet(GenerateCertsCmd, pflag.ContinueOnError)
	cmd.DefineFlags(flags, env.Build.Version())

	err := flags.Parse(env.Args)
	if err != nil {
		if !errors.Is(err, pflag.ErrHelp) {
			return err
		}

		return nil
	}

	_, span := telemetry.Start(context.Background(), "certs.generate")
	defer span.End()

	err = cmd.Execute()
	telemetry.Fail(span, err)

	return err
}
