package main

import (
	"github.com/charmbracelet/log"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/evg4b/uncors/internal/uncors"
	"github.com/muesli/termenv"
	"os"
)

func main() {
	infra.ConfigureLogger()
	log.SetColorProfile(termenv.TrueColor)
	log.SetLevel(log.DebugLevel)

	_, err := os.Stdout.Write([]byte{0x1B, 0x5B, 0x33, 0x3B, 0x4A, 0x1B, 0x5B, 0x48, 0x1B, 0x5B, 0x32, 0x4A})
	if err != nil {
		panic(err)
	}

	println(tui.Logo("X.X.X"))

	testLogger(log.Default())
	testLogger(uncors.NewCacheLogger(log.Default()))
	testLogger(uncors.NewProxyLogger(log.Default()))
	testLogger(uncors.NewMockLogger(log.Default()))
	testLogger(uncors.NewStaticLogger(log.Default()))
}

func testLogger(logger *log.Logger) {
	logger.Debug("Test debug")
	logger.With("key", "value").Debug("Test debug with key")
	logger.Info("Test info")
	logger.With("key", "value").Info("Test info with key")
	logger.Warn("Test warn")
	logger.With("key", "value").Warn("Test warn with key")
	logger.Error("Test error")
	logger.With("key", "value").Error("Test error with key")
}
