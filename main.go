package main

import (
	"github.com/evg4b/uncors/internal/tui"
	"os"

	"github.com/evg4b/uncors/internal/tui/styles"
	"github.com/muesli/termenv"

	"github.com/evg4b/uncors/internal/config/validators"

	"github.com/charmbracelet/log"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
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
	defer helpers.PanicInterceptor(func(value any) {
		log.Error(value)
		os.Exit(1)
	})

	pflag.Usage = func() {
		println(tui.Logo(Version))
		helpers.FPrintf(os.Stdout, "Usage of %s:\n", os.Args[0])
		pflag.PrintDefaults()
	}

	fs := afero.NewOsFs()

	viperInstance := viper.GetViper()

	log.SetReportTimestamp(false)
	log.SetReportCaller(false)
	log.SetStyles(&styles.DefaultStyles)
	log.SetColorProfile(termenv.ColorProfile())

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
	go func() {
		shutdownErr := helpers.GracefulShutdown(ctx, func(shutdownCtx context.Context) error {
			log.Debug("shutdown signal received")

			return app.Shutdown(shutdownCtx)
		})
		if shutdownErr != nil {
			panic(shutdownErr)
		}
	}()
	app.Wait()
	log.Info("Server was stopped")
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

	return uncorsConfig
}
