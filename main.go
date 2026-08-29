package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/di"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/evg4b/uncors/internal/uncors"
	uncorsapp "github.com/evg4b/uncors/internal/uncors_app"
	"github.com/spf13/afero"
	"github.com/spf13/pflag"
)

var Version = "v0.7.0"

const generateCertsCmd = "generate-certs"

func main() {
	exitCode := run()
	os.Exit(exitCode)
}

func run() int {
	fs := afero.NewOsFs()
	console := tui.NewCliOutput(os.Stdout)

	defer helpers.PanicInterceptor(func(value any) {
		console.Error(value)
		log.Fatalf("Caught panic: %v", value)
	})

	if len(os.Args) > 1 && os.Args[1] == generateCertsCmd {
		container := newContainer(fs, di.WithStdout(os.Stdout))
		defer container.Close()

		return runGenerateCerts(container)
	}

	pflag.Usage = func() {
		tui.PrintLogo(console, Version)
		fmt.Fprintf(console, "Usage of %s:\n", os.Args[0])
		pflag.PrintDefaults()
	}

	uncorsConfig, configPath, err := loadConfiguration(fs)
	if err != nil {
		panic(err)
	}

	// The output implementation is chosen before the container is built, so
	// nothing can capture the wrong one.
	if uncorsConfig.Interactive {
		return runInteractive(console, fs, configPath, uncorsConfig)
	}

	return runNonInteractive(context.Background(), fs, configPath, uncorsConfig)
}

func newContainer(fs afero.Fs, options ...di.ContainerOption) *di.Container {
	return di.NewContainer(append([]di.ContainerOption{
		di.WithFs(fs),
		di.WithVersion(Version),
	}, options...)...)
}

// runGenerateCerts executes the generate-certs sub-command and returns an exit code.
func runGenerateCerts(container *di.Container) int {
	cmd := container.GenerateCertsCommand()
	output := container.CliOutput()

	flags := pflag.NewFlagSet(generateCertsCmd, pflag.ContinueOnError)
	cmd.DefineFlags(flags)

	err := flags.Parse(os.Args[2:])
	if err != nil {
		output.Error(err)
		log.Printf("Error: %v", err)

		return 1
	}

	err = cmd.Execute()
	if err != nil {
		output.Error(err)
		log.Printf("Error: %v", err)

		return 1
	}

	return 0
}

// runNonInteractive starts the proxy in non-interactive (headless) mode and
// blocks until the server shuts down. The config file is watched for changes
// when configPath is non-empty.
func runNonInteractive(
	ctx context.Context,
	fs afero.Fs,
	configPath string,
	cfg *config.UncorsConfig,
) int {
	container := newContainer(fs, di.WithStdout(os.Stdout))
	defer container.Close()

	output := container.CliOutput()

	app := uncors.CreateUncors(container)

	// Headless mode has no TUI, so it must supply the sink that drains the
	// tracker itself; without a consumer the request path would only be able to
	// drop activity events.
	tracker := container.RequestTracker()

	go server.RequestPrinter(tracker, output)

	reloader := uncors.NewReloader(app, output, configLoader(fs), configPath)
	defer func() {
		closeErr := reloader.Close()
		if closeErr != nil {
			log.Printf("Failed to close config watcher: %v", closeErr)
		}
	}()

	err := app.Start(ctx, cfg)
	if err != nil {
		panic(err)
	}

	err = reloader.Start(ctx)
	if err != nil {
		log.Printf("Failed to start config watcher: %v", err)
		output.Errorf("Failed to start config watcher: %v", err)
	}

	go startVersionChecker(ctx, container, cfg.Proxy)

	go helpers.GracefulShutdown(ctx, func(shutdownCtx context.Context) error {
		log.Println("shutdown signal received")

		return app.Shutdown(shutdownCtx)
	})

	app.Wait()

	if dropped := tracker.Dropped(); dropped > 0 {
		output.Warnf("%d activity lines were dropped to keep the proxy responsive", dropped)
	}

	output.Info("Server was stopped")

	return 0
}

// startVersionChecker waits for a short delay then checks for a newer release.
func startVersionChecker(ctx context.Context, container *di.Container, proxy string) {
	const checkDelay = 50 * time.Millisecond

	time.Sleep(checkDelay)

	container.VersionChecker(proxy).
		CheckNewVersion(ctx)
}

// runInteractive starts the proxy in interactive TUI mode.
func runInteractive(console contracts.Output, fs afero.Fs, configPath string, cfg *config.UncorsConfig) int {
	output := uncorsapp.NewOutput()

	container := newContainer(fs, di.WithOutput(output))
	defer container.Close()

	app := uncorsapp.NewUncorsApp(
		container,
		output,
		configPath,
		cfg,
		configLoader(fs),
	)

	_, err := tea.NewProgram(app).Run()
	if err != nil {
		log.Printf("Interactive mode failed: %v", err)
		console.Error(err)

		return 1
	}

	return 0
}

const (
	logFileName  = "uncors.log"
	logFileFlags = os.O_CREATE | os.O_WRONLY | os.O_APPEND
	logFilePerm  = 0o644
)

// configLoader returns the loader both run modes use to reload the
// configuration. Errors are propagated so that a malformed config is reported
// as such instead of silently becoming a nil configuration.
func configLoader(fs afero.Fs) uncors.ConfigLoader {
	return func() (*config.UncorsConfig, error) {
		uncorsConfig, _, err := loadConfiguration(fs)

		return uncorsConfig, err
	}
}

// loadConfiguration loads and validates the configuration from CLI args and the
// config file.
func loadConfiguration(fs afero.Fs) (*config.UncorsConfig, string, error) {
	uncorsConfig, configPath, err := config.LoadConfiguration(fs, os.Args)
	if err != nil {
		return nil, "", err
	}

	if uncorsConfig.Debug {
		logFile, err := os.OpenFile(logFileName, logFileFlags, logFilePerm)
		if err != nil {
			return nil, "", fmt.Errorf("failed to open log file: %w", err)
		}

		log.SetOutput(logFile)
		log.Print("Enabled debug messages")
	} else {
		log.SetOutput(io.Discard)
	}

	return uncorsConfig, configPath, nil
}
