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
	"github.com/evg4b/uncors/internal/di"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/evg4b/uncors/internal/uncors"
	uncorsapp "github.com/evg4b/uncors/internal/uncors_app"
	"github.com/evg4b/uncors/internal/version"
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
	container := di.NewContainer(fs, os.Stdout)

	output := container.CliOutput()

	defer helpers.PanicInterceptor(func(value any) {
		output.Error(value)
		log.Fatalf("Caught panic: %v", value)
	})

	if len(os.Args) > 1 && os.Args[1] == generateCertsCmd {
		return runGenerateCerts(container)
	}

	pflag.Usage = func() {
		tui.PrintLogo(output, Version)
		fmt.Fprintf(output, "Usage of %s:\n", os.Args[0])
		pflag.PrintDefaults()
	}

	uncorsConfig, configPath := loadConfiguration(fs)

	if uncorsConfig.Interactive {
		return runInteractive(fs, configPath, uncorsConfig)
	}

	return runNonInteractive(context.Background(), container, configPath, uncorsConfig)
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
	container *di.Container,
	configPath string,
	cfg *config.UncorsConfig,
) int {
	output := container.CliOutput()

	app := uncors.CreateUncors(container.Fs(), container.Server(), output, Version)

	go server.RequestPrinter(container.RequestTracker(), output)

	if configPath != "" {
		startConfigWatcher(ctx, container, configPath, app)
	}

	err := app.Start(ctx, cfg)
	if err != nil {
		panic(err)
	}

	go startVersionChecker(ctx, output, cfg.Proxy)

	go helpers.GracefulShutdown(ctx, func(shutdownCtx context.Context) error {
		log.Println("shutdown signal received")

		return app.Shutdown(shutdownCtx)
	})

	app.Wait()
	output.Info("Server was stopped")

	return 0
}

// startConfigWatcher begins watching the config file and restarts the proxy on
// every change. The watcher lives for the process lifetime (not closed explicitly).
func startConfigWatcher(
	ctx context.Context,
	container *di.Container,
	configPath string,
	app *uncors.Uncors,
) {
	output := container.CliOutput()
	fs := container.Fs()
	watcher := config.NewWatcher(configPath)

	err := watcher.Watch(ctx, func() {
		defer helpers.PanicInterceptor(func(value any) {
			log.Printf("Config reloading error: %v", value)
			output.Errorf("Config reloading error: %v", value)
		})

		reloaded, _ := loadConfiguration(fs)

		restartErr := app.Restart(ctx, reloaded)
		if restartErr != nil {
			log.Printf("Failed to restart server: %v", restartErr)
			output.Errorf("Failed to restart server: %v", restartErr)
		}
	})
	if err != nil {
		log.Printf("Failed to start config watcher: %v", err)
		output.Errorf("Failed to start config watcher: %v", err)

		return
	}
}

// startVersionChecker waits for a short delay then checks for a newer release.
func startVersionChecker(ctx context.Context, output *tui.CliOutput, proxy string) {
	const checkDelay = 50 * time.Millisecond

	versionChecker := version.NewVersionChecker(
		version.WithOutput(output),
		version.WithHTTPClient(infra.MakeHTTPClient(proxy)),
		version.WithCurrentVersion(Version),
	)

	time.Sleep(checkDelay)
	versionChecker.CheckNewVersion(ctx)
}

// runInteractive starts the proxy in interactive TUI mode.
func runInteractive(fs afero.Fs, configPath string, cfg *config.UncorsConfig) int {
	app := uncorsapp.NewUncorsApp(
		Version,
		fs,
		configPath,
		cfg,
		func() *config.UncorsConfig {
			reloaded, _ := loadConfiguration(fs)

			return reloaded
		},
	)

	_, err := tea.NewProgram(app).Run()
	if err != nil {
		log.Fatal(err)
	}

	return 0
}

const (
	logFileName  = "uncors.log"
	logFileFlags = os.O_CREATE | os.O_WRONLY | os.O_APPEND
	logFilePerm  = 0o644
)

// loadConfiguration loads and validates the configuration from CLI args and the
// config file. It panics on any error so that the PanicInterceptor in run() can
// display a human-readable message and exit cleanly.
func loadConfiguration(fs afero.Fs) (*config.UncorsConfig, string) {
	uncorsConfig, configPath, err := config.LoadConfiguration(fs, os.Args)
	if err != nil {
		panic(err)
	}

	if uncorsConfig.Debug {
		logFile, err := os.OpenFile(logFileName, logFileFlags, logFilePerm)
		if err != nil {
			panic(fmt.Sprintf("Failed to open log file: %v", err))
		}

		log.SetOutput(logFile)
		log.Print("Enabled debug messages")
	} else {
		log.SetOutput(io.Discard)
	}

	return uncorsConfig, configPath
}
