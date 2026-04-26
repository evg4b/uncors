package main

import (
	"context"
	"fmt"
	"os"

	clog "github.com/charmbracelet/log"
	"github.com/evg4b/uncors/internal/commands"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/config/validators"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/log"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/evg4b/uncors/internal/uncors"
	"github.com/evg4b/uncors/internal/version"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/afero"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var Version = "X.X.X"

func main() {
	exitCode := run()
	os.Exit(exitCode)
}

func run() int {
	logger := log.Default()

	defer helpers.PanicInterceptor(func(value any) {
		logger.Error(value)
	})

	fs := afero.NewOsFs()

	infra.ConfigureLogger()

	clog.Error("test", "demo", 123)
	logger.Error("test", "demo", 123)
	clog.Infof("test %d: %d", 123, 123)
	logger.Infof("test %d: %d", 123, 123)

	os.Exit(0)

	if len(os.Args) > 1 && os.Args[1] == "generate-certs" {
		cmd := commands.NewGenerateCertsCommand(fs)
		flags := pflag.NewFlagSet("generate-certs", pflag.ExitOnError)
		cmd.DefineFlags(flags)

		err := flags.Parse(os.Args[2:])
		if err != nil {
			logger.Error(err)

			return 1
		}

		err = cmd.Execute()
		if err != nil {
			logger.Error(err)
			return 1
		}

		return 0
	}

	pflag.Usage = func() {
		tui.PrintLogo(os.Stdout, Version)
		fmt.Fprintf(os.Stdout, "Usage of %s:\n", os.Args[0])
		pflag.PrintDefaults()
	}

	viperInstance := viper.GetViper()

	uncorsConfig := loadConfiguration(logger, viperInstance, fs)

	ctx := context.Background()
	app := uncors.CreateUncors(fs, logger, Version)

	viperInstance.OnConfigChange(func(_ fsnotify.Event) {
		defer helpers.PanicInterceptor(func(value any) {
			logger.Errorf("Config reloading error: %v", value)
		})

		err := app.Restart(ctx, loadConfiguration(logger, viperInstance, fs))
		if err != nil {
			logger.Errorf("Failed to restart server: %v", err)
		}
	})
	viperInstance.WatchConfig()

	go version.CheckNewVersion(ctx, infra.MakeHTTPClient(uncorsConfig.Proxy), Version)

	err := app.Start(ctx, uncorsConfig)
	if err != nil {
		panic(err)
	}

	go helpers.GracefulShutdown(ctx, func(shutdownCtx context.Context) error {
		logger.Debug("shutdown signal received")

		return app.Shutdown(shutdownCtx)
	})

	app.Wait()
	logger.Info("Server was stopped")

	return 0
}

func loadConfiguration(logger *log.Logger, viperInstance *viper.Viper, fs afero.Fs) *config.UncorsConfig {
	uncorsConfig := config.LoadConfiguration(viperInstance, os.Args)

	err := validators.ValidateConfig(uncorsConfig, fs)
	if err != nil {
		panic(err)
	}

	if uncorsConfig.Debug {
		logger.SetLevel(log.DebugLevel)
		logger.Debug("Enabled debug messages")
	} else {
		logger.SetLevel(log.InfoLevel)
	}

	return uncorsConfig
}
