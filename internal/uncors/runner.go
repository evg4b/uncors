package uncors

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
)

// shutdownTimeout bounds how long in-flight requests are given to finish. It is
// a property of the proxy, not of the run mode that started it.
const shutdownTimeout = 15 * time.Second

// VersionCheck looks for a newer release and reports it to the user.
type VersionCheck = func(ctx context.Context)

// Runner owns the process wide lifecycle of the proxy: startup, config watching,
// the version check, signal handling and shutdown.
//
// Both run modes drive the same Runner, so none of that can differ between an
// interactive and a headless process — which is how interactive mode came to
// ignore SIGTERM, headless came to leak its config watcher, and the two came to
// use different shutdown timeouts.
type Runner struct {
	app          *Uncors
	reloader     *Reloader
	output       contracts.Output
	checkVersion VersionCheck
}

func NewRunner(app *Uncors, reloader *Reloader, output contracts.Output, checkVersion VersionCheck) *Runner {
	return &Runner{
		app:          app,
		reloader:     reloader,
		output:       output,
		checkVersion: checkVersion,
	}
}

// Start brings the proxy up and begins watching the config file and the shutdown
// signals. It returns once the listeners are live.
func (r *Runner) Start(ctx context.Context, uncorsConfig *config.UncorsConfig) error {
	err := r.app.Start(ctx, uncorsConfig)
	if err != nil {
		return err
	}

	watchErr := r.reloader.Start(ctx)
	if watchErr != nil {
		slog.Error("failed to watch the config file", "err", watchErr)
		r.output.Errorf("Failed to watch config file: %v", watchErr)
	}

	go r.awaitShutdownSignal(ctx)

	if r.checkVersion != nil {
		go r.checkVersion(ctx)
	}

	return nil
}

// Wait blocks until every listener has stopped.
func (r *Runner) Wait() {
	r.app.Wait()
}

// Reload applies the configuration currently on disk.
func (r *Runner) Reload(ctx context.Context) error {
	return r.reloader.Reload(ctx)
}

// Shutdown stops watching the config and drains the listeners.
func (r *Runner) Shutdown(ctx context.Context) error {
	// The shutdown deadline must survive the cancellation that usually triggers
	// it, or in-flight requests would be cut off instead of drained.
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()

	return errors.Join(
		r.reloader.Close(),
		r.app.Shutdown(shutdownCtx),
	)
}

func (r *Runner) awaitShutdownSignal(ctx context.Context) {
	GracefulShutdown(ctx, func(shutdownCtx context.Context) error {
		slog.Info("shutdown signal received")

		return r.Shutdown(shutdownCtx)
	})
}
