package main

import (
	"context"
	"os"
	"time"

	"github.com/evg4b/uncors/internal/cli"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/telemetry"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/spf13/afero"
)

var Version = "v0.0.0"

const flushTimeout = 5 * time.Second

func main() {
	ctx := context.Background()
	flush := flusher(telemetry.Setup(ctx, Version))

	defer helpers.PanicInterceptor(func(value any) {
		flush()
		report(value)
		osExit(1)
	})

	// handleError exits without running defers, so the exporters are flushed first.
	err := run(ctx)

	flush()
	handleError(err)
}

func flusher(shutdown func(context.Context) error) func() {
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), flushTimeout)
		defer cancel()

		_ = shutdown(ctx)
	}
}

func run(ctx context.Context) error {
	env := cli.Env{
		Fs:      afero.NewOsFs(),
		Stdout:  os.Stdout,
		Args:    os.Args[1:],
		Version: Version,
	}

	if len(os.Args) >= 2 && os.Args[1] == cli.GenerateCertsCmd {
		env.Args = os.Args[2:]

		return cli.GenerateCerts(env)
	}

	return cli.RunUncors(ctx, env)
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
