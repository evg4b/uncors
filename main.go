package main

import (
	"context"
	"os"

	"github.com/evg4b/uncors/internal/cli"
	"github.com/evg4b/uncors/internal/di"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/spf13/afero"
)

const Version = "v0.0.0"

func main() {
	infra.SetupLogging()

	container := di.NewContainer(
		di.WithFs(afero.NewOsFs()),
		di.WithStdout(os.Stdout),
		di.WithVersion(Version),
	)

	defer func() {
		handleError(container.Close())
	}()

	if len(os.Args) >= 2 && os.Args[1] == cli.GenerateCertsCmd {
		container.Override(di.WithArgs(os.Args[2:]))

		err := cli.GenerateCerts(container)
		handleError(err)

		return
	}

	container.Override(di.WithArgs(os.Args[1:]))
	err := cli.RunUncors(context.Background(), container)
	handleError(err)
}

func handleError(err error) {
	if err != nil {
		tui.NewCliOutput(os.Stdout).
			Error(err)

		os.Exit(1)
	}
}
