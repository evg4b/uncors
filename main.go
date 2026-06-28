package main

import (
	"context"
	"os"

	"github.com/evg4b/uncors/internal/cli"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/spf13/afero"
)

func main() {
	infra.SetupLogging()

	if len(os.Args) >= 2 && os.Args[1] == cli.GenerateCertsCmd {
		err := cli.GenerateCerts(os.Args[2:])
		if err != nil {
			tui.NewCliOutput(os.Stdout).
				Error(err)
		}

		return
	}

	err := cli.RunUncors(context.Background(), afero.NewOsFs(), os.Args[1:])
	if err != nil {
		tui.NewCliOutput(os.Stdout).
			Error(err)
	}
}
