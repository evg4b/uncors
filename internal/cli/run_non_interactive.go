package cli

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/di"
	"github.com/evg4b/uncors/internal/server"
)

const (
	shutdownTimeout   = 15 * time.Second
	versionCheckDelay = 50 * time.Millisecond
)

// runNonInteractive starts the proxy in headless mode and blocks until the
// server stops, either on a shutdown signal or when ctx is cancelled.
func runNonInteractive(
	ctx context.Context,
	container *di.Container,
	cfg *config.UncorsConfig,
	cfgPath string,
) error {
	output := container.CliOutput()
	proxy := container.Proxy()

	// Headless mode has no TUI, so it must drain the request tracker itself;
	// without a consumer the request path can only drop activity events.
	tracker := container.RequestTracker()
	go server.RequestPrinter(tracker, output)

	err := proxy.Start(ctx, cfg)
	if err != nil {
		return err
	}

	go versionCheck(ctx, container, cfg.Proxy)
	go watchConfig(ctx, container, proxy, cfgPath)
	go awaitShutdown(ctx, proxy)

	proxy.Wait()

	if dropped := tracker.Dropped(); dropped > 0 {
		output.Warnf("%d activity lines were dropped to keep the proxy responsive", dropped)
	}

	output.Info("Server was stopped")

	return nil
}

// awaitShutdown stops the proxy on the first OS signal or context cancellation.
// The shutdown itself runs on a fresh context because ctx may be the one that
// triggered it.
func awaitShutdown(ctx context.Context, proxy *di.Proxy) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	defer signal.Stop(stop)

	select {
	case sig := <-stop:
		if sig == syscall.SIGINT {
			// Move past the "^C" the terminal echoed.
			_, _ = os.Stdout.WriteString("\n")
		}

		log.Println("shutdown signal received")
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()

	_ = proxy.Shutdown(shutdownCtx)
}

// watchConfig restarts the proxy on every change to the config file. It is a
// no-op when no config file is in use.
func watchConfig(ctx context.Context, container *di.Container, proxy *di.Proxy, cfgPath string) {
	output := container.CliOutput()

	watcher := config.NewWatcher(cfgPath)

	err := watcher.Watch(ctx, func() {
		reloaded, _, err := config.LoadConfiguration(container.Fs(), container.Version(), container.Args())
		if err != nil {
			output.Error(err)

			return
		}

		err = proxy.Restart(ctx, reloaded)
		if err != nil {
			output.Error(err)
		}
	})
	if err != nil {
		output.Error(err)

		return
	}

	<-ctx.Done()

	_ = watcher.Close()
}

// versionCheck waits for a short delay then checks for a newer release.
func versionCheck(ctx context.Context, container *di.Container, proxy string) {
	time.Sleep(versionCheckDelay)

	container.VersionChecker(proxy).
		CheckNewVersion(ctx)
}
