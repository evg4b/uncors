package main

import (
	"os"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/log"
	"github.com/evg4b/uncors/internal/ui"
	"github.com/evg4b/uncors/internal/uncors"
	"github.com/evg4b/uncors/internal/version"
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
		ui.Logo(Version)
		helpers.Fprintf(os.Stdout, "Usage of %s:\n", os.Args[0])
		pflag.PrintDefaults()
	}

	viperInstance := viper.GetViper()
	uncorsConfig := loadConfiguration(viperInstance)

	fs := afero.NewOsFs()
	ctx := context.Background()
	app := uncors.CreateApp(fs, Version)
	viperInstance.OnConfigChange(func(in fsnotify.Event) {
		app.Restart(ctx, loadConfiguration(viperInstance))
	})
	viperInstance.WatchConfig()
	go version.CheckNewVersion(ctx, infra.MakeHTTPClient(uncorsConfig.Proxy), Version)
	app.Start(ctx, uncorsConfig)
	go uncors.GracefulShutdown(ctx, 5*time.Second, app.Shutdown)
	app.Wait()
}

func loadConfiguration(viperInstance *viper.Viper) *config.UncorsConfig {
	uncorsConfig := config.LoadConfiguration(viperInstance, os.Args)
	if uncorsConfig.Debug {
		log.EnableDebugMessages()
		log.Debug("Enabled debug messages")
	} else {
		log.DisableDebugMessages()
	}

	return uncorsConfig
}
