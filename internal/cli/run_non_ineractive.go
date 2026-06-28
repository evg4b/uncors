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
	"github.com/evg4b/uncors/internal/tui"
)

const shutdownTimeout = 15 * time.Second

func runNonIneractive(
	ctx context.Context,
	container *di.Container,
	cfg *config.UncorsConfig,
	cfgPath string,
	args []string,
) error {
	output := container.CliOutput()
	tui.PrintLogo(output, container.Version())
	output.Print("")
	output.WarnBox(tui.DisclaimerMessage)
	output.Print("")
	output.InfoBox(cfg.Mappings.String())
	output.Print("")

	targets, err := container.Targets(cfg)
	if err != nil {
		return err
	}

	srv := container.Server()

	err = srv.Start(ctx, targets)
	if err != nil {
		return err
	}

	go startVersionChecker(ctx, container, cfg.Proxy)

	go func() {
		watcher := config.NewWatcher(cfgPath)

		err := watcher.Watch(ctx, func() { reloadServer(ctx, container, srv, args) })
		if err != nil {
			output.Error(err)
		}
	}()

	go func() { //nolint:gosec // G118: shutdown needs a fresh context because parent ctx is being cancelled
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

		defer signal.Stop(stop)

		select {
		case sig := <-stop:
			if sig == syscall.SIGINT {
				_, _ = os.Stdout.WriteString("\n")
			}

			log.Println("shutdown signal received")
		case <-ctx.Done():
		}

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		_ = srv.Shutdown(shutdownCtx)
	}()

	srv.Wait()
	output.Info("Server was stopped")

	return nil
}

func reloadServer(ctx context.Context, container *di.Container, srv *server.Server, args []string) {
	output := container.CliOutput()

	newUncorsConfig, _, err := config.LoadConfiguration(container.Fs(), args)
	if err != nil {
		output.Error(err)

		return
	}

	output.Info("Restarting server....")

	targets, err := container.Targets(newUncorsConfig)
	if err != nil {
		output.Error(err)

		return
	}

	err = srv.Restart(ctx, targets)
	if err != nil {
		output.Error(err)

		return
	}

	output.InfoBox(
		"Server restarted",
		newUncorsConfig.Mappings.String(),
	)
}

// startVersionChecker waits for a short delay then checks for a newer release.
func startVersionChecker(ctx context.Context, container *di.Container, proxy string) {
	const checkDelay = 50 * time.Millisecond

	time.Sleep(checkDelay)

	container.VersionChecker(proxy).
		CheckNewVersion(ctx)
}
