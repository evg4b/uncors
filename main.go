package main

import (
	"context"
	"os"

	"github.com/evg4b/uncors/internal/cli"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/logging"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/spf13/afero"
)

var Version = "v0.0.0"

func main() {
	fs := afero.NewOsFs()

	logger, logFile := logging.Setup(fs, os.Getenv(logging.EnvVar))
	defer helpers.CloseSafe(logFile)

	env := cli.Env{
		Fs:      fs,
		Stdout:  os.Stdout,
		Args:    os.Args[1:],
		Version: Version,
		Logger:  logger,
	}

	// A panic anywhere below is a bug, but the user still deserves a readable
	// message instead of a raw stack trace.
	defer helpers.PanicInterceptor(func(value any) {
		report(value)
		osExit(1)
	})

	if len(os.Args) >= 2 && os.Args[1] == cli.GenerateCertsCmd {
		env.Args = os.Args[2:]

		handleError(cli.GenerateCerts(env))

		return
	}

	handleError(cli.RunUncors(context.Background(), env))
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
