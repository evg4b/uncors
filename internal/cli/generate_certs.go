package cli

import (
	"errors"

	"github.com/evg4b/uncors/internal/di"
	"github.com/spf13/pflag"
)

const GenerateCertsCmd = "generate-certs"

func GenerateCerts(container *di.Container) error {
	cmd := container.GenerateCertsCommand()

	flags := pflag.NewFlagSet(GenerateCertsCmd, pflag.ContinueOnError)
	cmd.DefineFlags(flags, container.Version())

	err := flags.Parse(container.Args())
	if err != nil {
		if !errors.Is(err, pflag.ErrHelp) {
			return err
		}

		return nil
	}

	return cmd.Execute()
}
