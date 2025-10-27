package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/evg4b/uncors/internal/commands"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/config/validators"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/evg4b/uncors/internal/uncors"
	"github.com/evg4b/uncors/internal/version"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/afero"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
)

var Version = "X.X.X"

func main() {
	exitCode := run()
	os.Exit(exitCode)
}

func run() int {
	defer helpers.PanicInterceptor(func(value any) {
		log.Error(value)
	})

	fs := afero.NewOsFs()
	infra.ConfigureLogger()

	if len(os.Args) > 1 && os.Args[1] == "generate-certs" {
		cmd := commands.NewGenerateCertsCommand(fs)
		flags := pflag.NewFlagSet("generate-certs", pflag.ExitOnError)
		cmd.DefineFlags(flags)
		if err := flags.Parse(os.Args[2:]); err != nil {
			log.Error(err)

			return 1
		}
		if err := cmd.Execute(); err != nil {
			log.Error(err)

			return 1
		}

		return 0
	}

	pflag.Usage = func() {
		println(tui.Logo(Version)) //nolint:forbidigo
		fmt.Fprintf(os.Stdout, "Usage of %s:\n", os.Args[0])
		pflag.PrintDefaults()
	}

	viperInstance := viper.GetViper()

	uncorsConfig := loadConfiguration(viperInstance, fs)

	ctx := context.Background()
	app := uncors.CreateApp(fs, log.Default(), Version)
	viperInstance.OnConfigChange(func(_ fsnotify.Event) {
		defer helpers.PanicInterceptor(func(value any) {
			log.Errorf("Config reloading error: %v", value)
		})

		app.Restart(ctx, loadConfiguration(viperInstance, fs))
	})
	viperInstance.WatchConfig()
	go version.CheckNewVersion(ctx, infra.MakeHTTPClient(uncorsConfig.Proxy), Version)
	app.Start(ctx, uncorsConfig)
	go helpers.GracefulShutdown(ctx, func(shutdownCtx context.Context) error {
		log.Debug("shutdown signal received")

		return app.Shutdown(shutdownCtx)
	})
	app.Wait()
	log.Info("Server was stopped")

	return 0
}

func loadConfiguration(viperInstance *viper.Viper, fs afero.Fs) *config.UncorsConfig {
	uncorsConfig := config.LoadConfiguration(viperInstance, os.Args)
	err := validators.ValidateConfig(uncorsConfig, fs)
	if err != nil {
		panic(err)
	}

	if uncorsConfig.Debug {
		log.SetLevel(log.DebugLevel)
		log.Debug("Enabled debug messages")
	} else {
		log.SetLevel(log.InfoLevel)
	}

	// Validate that hosts from mappings are present in the hosts file
	validators.ValidateHostsFileEntries(uncorsConfig)

	return uncorsConfig
}
