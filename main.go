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
	logFileFlags = os.O_CREATE | os.O_WRONLY | os.O_APPEND
	logFilePerm  = 0o644
)

func setupLogging() {
	path := os.Getenv("UNCORS_LOGGING")
	if path == "" {
		log.SetOutput(io.Discard)

		return
	}

	f, err := os.OpenFile(path, logFileFlags, logFilePerm)
	if err != nil {
		log.SetOutput(io.Discard)

		return
	}

	log.SetOutput(f)
}

func main() {
	setupLogging()

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
