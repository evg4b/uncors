package cli

import (
	"errors"
	"os"

	"github.com/evg4b/uncors/internal/di"
	"github.com/spf13/afero"
	"github.com/spf13/pflag"
)

const GenerateCertsCmd = "generate-certs"

func GenerateCerts(args []string) error {
	fs := afero.NewOsFs()

	container := di.NewContainer(
		di.WithFs(fs),
		di.WithStdout(os.Stdout),
		di.WithVersion(Version),
	)
	defer container.Close()

	cmd := container.GenerateCertsCommand()

	flags := pflag.NewFlagSet(GenerateCertsCmd, pflag.ContinueOnError)
	cmd.DefineFlags(flags)

	err := flags.Parse(args)
	if err != nil {
		if !errors.Is(err, pflag.ErrHelp) {
			return err
		}
		return nil
	}

	return cmd.Execute()
}
