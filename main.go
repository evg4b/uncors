package main

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/evg4b/uncors/internal/cli"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/spf13/afero"
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

	logFile, err := os.OpenFile(filepath.Clean(path), logFileFlags, logFilePerm)
	if err != nil {
		log.SetOutput(io.Discard)

		return
	}

	log.SetOutput(logFile)
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

	err := cli.RunUncors(context.Background(), afero.NewOsFs(), os.Args[1:])
	if err != nil {
		output.Error(err)
	}
}
