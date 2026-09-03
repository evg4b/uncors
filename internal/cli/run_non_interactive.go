package cli

import (
	"context"

	"github.com/evg4b/uncors/internal/app"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/di"
	"github.com/evg4b/uncors/internal/server"
)

// runNonInteractive starts the proxy in headless mode and blocks until the
// server stops, either on a shutdown signal or when ctx is cancelled. It drives
// the same app.Service the interactive mode does; the only difference is that
// here the events are rendered straight to the console.
func runNonInteractive(
	ctx context.Context,
	container *di.Container,
	cfg *config.UncorsConfig,
	cfgPath string,
) error {
	output := container.CliOutput()

	// Headless mode has no TUI, so it must drain the request tracker itself;
	// without a consumer the request path can only drop activity events.
	tracker := container.RequestTracker()
	go server.RequestPrinter(tracker, output)

	service := app.New(container, cfg, cfgPath, configLoader(container))
	defer func() { _ = service.Close() }()

	err := service.Run(ctx)
	if err != nil {
		return err
	}

	if dropped := tracker.Dropped(); dropped > 0 {
		output.Warnf("%d activity lines were dropped to keep the proxy responsive", dropped)
	}

	output.Info("Server was stopped")

	return nil
}
