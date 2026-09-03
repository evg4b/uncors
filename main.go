package main

import (
	"context"
	"os"

	"github.com/evg4b/uncors/internal/cli"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/di"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/spf13/afero"
)

var Version = "v0.0.0"

func main() {
	infra.SetupLogging()

	container := di.NewContainer(
		di.WithFs(afero.NewOsFs()),
		di.WithStdout(os.Stdout),
		di.WithVersion(Version),
		di.WithArgs(os.Args[1:]),
		// The composition root decides how the application talks to the user.
		// The container itself knows nothing about terminals.
		di.WithCliOutput(func() contracts.Output {
			return tui.NewCliOutput(os.Stdout)
		}),
	)

	// A panic anywhere below is a bug, but the user still deserves a readable
	// message instead of a raw stack trace.
	defer helpers.PanicInterceptor(func(value any) {
		report(value)
		osExit(1)
	})

	defer func() { handleError(container.Close()) }()

	if len(os.Args) >= 2 && os.Args[1] == cli.GenerateCertsCmd {
		container.Override(di.WithArgs(os.Args[2:]))

		handleError(cli.GenerateCerts(container))

		return
	}

	handleError(cli.RunUncors(context.Background(), container))
}

var osExit = os.Exit

func handleError(err error) {
	if err != nil {
		report(err)
		osExit(1)
	}
}

func report(value any) {
	tui.NewCliOutput(os.Stdout).
		Error(value)
}
