package cli

import (
	"context"

	"github.com/evg4b/uncors/internal/app"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/di"
	"github.com/evg4b/uncors/internal/render"
	"github.com/evg4b/uncors/internal/server"
)

// runNonInteractive starts the proxy in headless mode and blocks until the
// server stops, either on a shutdown signal or when ctx is cancelled. It drives
// the same app.Service interactive mode does; the only difference is that here
// the events are rendered straight to the console.
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

	// The renderer has to be draining before the service starts, or the startup
	// banner has nowhere to go.
	renderer := render.New(output, container.Version())
	go renderer.Consume(service.Events())

	err := service.Run(ctx)
	if err != nil {
		return err
	}

	reportDropped(output, tracker.Dropped(), "activity lines")
	reportDropped(output, service.DroppedEvents(), "service events")

	output.Info("Server was stopped")

	return nil
}

func reportDropped(output contracts.WarnOutput, dropped uint64, what string) {
	if dropped > 0 {
		output.Warnf("%d %s were dropped to keep the proxy responsive", dropped, what)
	}
}
