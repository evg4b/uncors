package cli

import (
	"context"
	"errors"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/di"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/spf13/afero"
)

func runNonIneractive(fs afero.Fs, uncorsConfig *config.UncorsConfig, configPath string, args []string) error {
	container := di.NewContainer(
		di.WithFs(fs),
		di.WithStdout(os.Stdout),
		// di.WithVersion("Version"),
	)
	defer container.Close()

	output := container.CliOutput()
	tui.PrintLogo(output, container.Version())
	output.Print("")
	output.WarnBox(tui.DisclaimerMessage)
	output.Print("")
	output.InfoBox(uncorsConfig.Mappings.String())
	output.Print("")

	targets, err := mappingsToTarget(container, uncorsConfig)
	if err != nil {
		return err
	}

	ctx := context.Background()

	server := container.Server()

	err = server.Start(ctx, targets)
	if err != nil {
		return err
	}

	go startVersionChecker(ctx, container, uncorsConfig.Proxy)

	go func(configPath string) {
		watcher := config.NewWatcher(configPath)

		err := watcher.Watch(ctx, func() { reloadServer(ctx, container, server, args) })
		if err != nil {
			output.Error(err)
		}
	}(configPath)

	go helpers.GracefulShutdown(ctx, func(shutdownCtx context.Context) error {
		log.Println("shutdown signal received")

		return server.Shutdown(shutdownCtx)
	})

	server.Wait()
	output.Info("Server was stopped")

	return nil
}

func reloadServer(ctx context.Context, container *di.Container, server *server.Server, args []string) {
	output := container.CliOutput()

	newUncorsConfig, _, err := config.LoadConfiguration(container.Fs(), args)
	if err != nil {
		output.Error(err)

		return
	}

	output.Info("Restarting server....")

	targets, err := mappingsToTarget(container, newUncorsConfig)
	if err != nil {
		output.Error(err)

		return
	}

	err = server.Restart(ctx, targets)
	if err != nil {
		output.Error(err)

		return
	}

	output.InfoBox(
		"Server restarted",
		newUncorsConfig.Mappings.String(),
	)
}

func mappingsToTarget(container *di.Container, uncorsConfig *config.UncorsConfig) ([]server.Target, error) {
	groupedMappings := uncorsConfig.Mappings.GroupByPort()
	targets := make([]server.Target, 0, len(groupedMappings))
	errs := make([]error, 0, len(groupedMappings))

	for _, group := range groupedMappings {
		muxRouter, err := container.Router(group.Mappings, &uncorsConfig.CacheConfig, uncorsConfig.Proxy)
		if err != nil {
			errs = append(errs, err)

			continue
		}

		targets = append(targets, server.Target{
			Address:   net.JoinHostPort(baseAddress, strconv.Itoa(group.Port)),
			Handler:   muxRouter,
			EnableTLS: group.Scheme == "https",
		})
	}

	return targets, errors.Join(errs...)
}

// startVersionChecker waits for a short delay then checks for a newer release.
func startVersionChecker(ctx context.Context, container *di.Container, proxy string) {
	const checkDelay = 50 * time.Millisecond

	time.Sleep(checkDelay)

	container.VersionChecker(proxy).
		CheckNewVersion(ctx)
}
