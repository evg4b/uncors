package main

import (
	"io"
	"log"
	"os"

	"github.com/evg4b/uncors/internal/cli"
	"github.com/evg4b/uncors/internal/tui"
)

var Version = "v0.7.0"

const (
	logFileName  = "uncors.log"
	logFileFlags = os.O_CREATE | os.O_WRONLY | os.O_APPEND
	logFilePerm  = 0o644
)

func main() {
	log.SetOutput(io.Discard)

	output := tui.NewCliOutput(os.Stdout)

	if len(os.Args) >= 2 && os.Args[1] == cli.GenerateCertsCmd {
		err := cli.GenerateCerts(os.Args[2:])
		if err != nil {
			output.Error(err)
		}

		return
	}

	err := cli.RunUncors(os.Args[1:])
	if err != nil {
		output.Error(err)
	}
}
