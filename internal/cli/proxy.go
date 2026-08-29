package cli

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/di"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/internal/uncors"
	uncorsapp "github.com/evg4b/uncors/internal/uncors_app"
	"github.com/spf13/afero"
	"github.com/spf13/pflag"
)

const (
	logFileName  = "uncors.log"
	logFileFlags = os.O_CREATE | os.O_WRONLY | os.O_APPEND
	logFilePerm  = 0o644
)

// proxyCommand is the root command: it runs the proxy itself.
func proxyCommand() Command {
	var flags *config.Flags

	return Command{
		Short: "Run the proxy (default)",
		Flags: func(set *pflag.FlagSet) {
			flags = config.DefineFlags(set)
		},
		Run: func(ctx context.Context, env Env, _ *pflag.FlagSet) error {
			uncorsConfig, err := loadConfiguration(env.Fs, flags)
			if err != nil {
				return err
			}

			// The output implementation is chosen before the container is built,
			// so that nothing can capture the wrong one.
			if uncorsConfig.Interactive && isTerminal(env.Stdout) {
				return runInteractive(env, flags, uncorsConfig)
			}

			return runHeadless(ctx, env, flags, uncorsConfig)
		},
	}
}

// isTerminal reports whether the writer is an interactive terminal. Interactive
// mode is the default, so without this check `uncors | tee log.txt`, a CI script
// and the Docker image would all start a full-screen TUI against a pipe.
func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeCharDevice != 0
}

// runHeadless starts the proxy without a terminal UI and blocks until the server
// shuts down.
func runHeadless(ctx context.Context, env Env, flags *config.Flags, cfg *config.UncorsConfig) error {
	container := newContainer(env, di.WithStdout(env.Stdout))
	defer container.Close()

	output := container.CliOutput()

	app := uncors.CreateUncors(container)

	// Headless mode has no TUI, so it must supply the sink that drains the
	// tracker itself; without a consumer the request path would only be able to
	// drop activity events.
	tracker := container.RequestTracker()

	go server.RequestPrinter(tracker, output)

	runner := uncors.NewRunner(
		app,
		uncors.NewReloader(app, output, configLoader(env.Fs, flags), flags.ConfigPath()),
		output,
		versionCheck(container, cfg.Proxy),
	)

	err := runner.Start(ctx, cfg)
	if err != nil {
		return err
	}

	runner.Wait()

	if dropped := tracker.Dropped(); dropped > 0 {
		output.Warnf("%d activity lines were dropped to keep the proxy responsive", dropped)
	}

	output.Info("Server was stopped")

	return nil
}

// runInteractive starts the proxy with the terminal UI.
func runInteractive(env Env, flags *config.Flags, cfg *config.UncorsConfig) error {
	output := uncorsapp.NewOutput()

	container := newContainer(env, di.WithOutput(output))
	defer container.Close()

	app := uncorsapp.NewUncorsApp(
		container,
		output,
		flags.ConfigPath(),
		cfg,
		configLoader(env.Fs, flags),
		versionCheck(container, cfg.Proxy),
	)

	_, err := tea.NewProgram(app).Run()
	if err != nil {
		return fmt.Errorf("interactive mode failed: %w", err)
	}

	return nil
}

func newContainer(env Env, options ...di.ContainerOption) *di.Container {
	return di.NewContainer(append([]di.ContainerOption{
		di.WithFs(env.Fs),
		di.WithVersion(env.Version),
	}, options...)...)
}

// versionCheck builds the release check both run modes perform after startup.
func versionCheck(container *di.Container, proxy string) uncors.VersionCheck {
	return func(ctx context.Context) {
		checker, err := container.VersionChecker(proxy)
		if err != nil {
			log.Printf("Version check failed: %v", err)

			return
		}

		checker.CheckNewVersion(ctx)
	}
}

// configLoader returns the loader both run modes use to reload the
// configuration. The flags were parsed once at startup and are re-applied as
// they are, so saving the config file re-reads only the file.
func configLoader(fs afero.Fs, flags *config.Flags) uncors.ConfigLoader {
	return func() (*config.UncorsConfig, error) {
		return loadConfiguration(fs, flags)
	}
}

func loadConfiguration(fs afero.Fs, flags *config.Flags) (*config.UncorsConfig, error) {
	uncorsConfig, err := config.LoadConfiguration(fs, flags)
	if err != nil {
		return nil, err
	}

	err = configureLogging(uncorsConfig.Debug)
	if err != nil {
		return nil, err
	}

	return uncorsConfig, nil
}

func configureLogging(debug bool) error {
	if !debug {
		log.SetOutput(io.Discard)

		return nil
	}

	logFile, err := os.OpenFile(logFileName, logFileFlags, logFilePerm)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	log.SetOutput(logFile)
	log.Print("Enabled debug messages")

	return nil
}
