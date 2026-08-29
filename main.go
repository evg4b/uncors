package main

import (
	"context"
	"os"

	"github.com/evg4b/uncors/internal/cli"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/spf13/afero"
)

var Version = "v0.7.0"

func main() {
	os.Exit(cli.Execute(context.Background(), cli.Env{
		Fs:      afero.NewOsFs(),
		Stdout:  os.Stdout,
		Console: tui.NewCliOutput(os.Stdout),
		Version: Version,
	}, os.Args[1:]))
}
