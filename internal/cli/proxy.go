package cli

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/di"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/server"
	tuiapp "github.com/evg4b/uncors/internal/tui/app"
	"github.com/evg4b/uncors/internal/uncors"
	"github.com/spf13/afero"
	"github.com/spf13/pflag"
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
			interactive := uncorsConfig.Interactive && isTerminal(env.Stdout)

			closer, err := setupLogging(flags, interactive)
			if err != nil {
				return err
			}

			if closer != nil {
				defer closer.Close()
			}

			if interactive {
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

// setupLogging installs the process logger. In interactive mode the diagnostics
// must not be written to the terminal, where they would corrupt the alt-screen,
// so they are routed into the history view instead — unless the user asked for a
// log file, which is a request to keep them out of the view.
func setupLogging(flags *config.Flags, interactive bool) (io.Closer, error) {
	options := infra.LogOptions{Level: flags.LogLevel(), File: flags.LogFile()}

	if interactive && options.File == "" {
		level, err := infra.ParseLogLevel(options.Level)
		if err != nil {
			return nil, err
		}

		options.Handler = tuiapp.NewLogHandler(level)
	}

	return infra.SetupLogging(options)
}

// runHeadless starts the proxy without a terminal UI and blocks until the server
// shuts down.
func runHeadless(ctx context.Context, env Env, flags *config.Flags, cfg *config.UncorsConfig) error {
	container := newContainer(env, flags, di.WithStdout(env.Stdout))
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
	output := tuiapp.NewOutput()

	container := newContainer(env, flags, di.WithOutput(output))
	defer container.Close()

	app := tuiapp.NewUncorsApp(
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

func newContainer(env Env, flags *config.Flags, options ...di.ContainerOption) *di.Container {
	return di.NewContainer(append([]di.ContainerOption{
		di.WithFs(env.Fs),
		di.WithVersion(env.Version),
		di.WithCADir(flags.CADir()),
	}, options...)...)
}

// versionCheck builds the release check both run modes perform after startup.
func versionCheck(container *di.Container, proxy string) uncors.VersionCheck {
	return func(ctx context.Context) {
		checker, err := container.VersionChecker(proxy)
		if err != nil {
			slog.Error("version check failed", "err", err)

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
	return config.LoadConfiguration(fs, flags)
}
