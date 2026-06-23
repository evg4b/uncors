package infra

import (
	"io"
	"log"
	"os"
	"path/filepath"
)

const (
	logFileFlags = os.O_CREATE | os.O_WRONLY | os.O_APPEND
	logFilePerm  = 0o644
)

func SetupLogging() {
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
