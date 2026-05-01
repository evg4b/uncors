package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/evg4b/uncors/internal/commands"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/config/validators"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/evg4b/uncors/internal/uncors"
	uncorsapp "github.com/evg4b/uncors/internal/uncors_app"
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
	output := tui.NewCliOutput(os.Stdout)

	defer helpers.PanicInterceptor(func(value any) {
		output.Error(value)
		log.Fatalf("Caught panic: %v", value)
	})

	fs := afero.NewOsFs()

	if len(os.Args) > 1 && os.Args[1] == "generate-certs" {
		cmd := commands.NewGenerateCertsCommand(
			commands.WithFs(fs),
			commands.WithOutput(output),
		)
		flags := pflag.NewFlagSet("generate-certs", pflag.ExitOnError)
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

	pflag.Usage = func() {
		tui.PrintLogo(output, Version)
		fmt.Fprintf(output, "Usage of %s:\n", os.Args[0])
		pflag.PrintDefaults()
	}

	viperInstance := viper.GetViper()

	uncorsConfig := loadConfiguration(viperInstance, fs)

	ctx := context.Background()

	if !uncorsConfig.Interactive {
		app := uncors.CreateUncors(fs, output, Version)

		viperInstance.OnConfigChange(func(_ fsnotify.Event) {
			defer helpers.PanicInterceptor(func(value any) {
				log.Printf("Config reloading error: %v", value)
				output.Errorf("Config reloading error: %v", value)
			})

			err := app.Restart(ctx, loadConfiguration(viperInstance, fs))
			if err != nil {
				log.Printf("Failed to restart server: %v", err)
				output.Errorf("Failed to restart server: %v", err)
			}
		})
		viperInstance.WatchConfig()

		err := app.Start(ctx, uncorsConfig)
		if err != nil {
			panic(err)
		}

		go func() {
			const checkDelay = 50 * time.Second

			versionChecker := version.NewVersionChecker(
				version.WithOutput(output),
				version.WithHTTPClient(infra.MakeHTTPClient(uncorsConfig.Proxy)),
				version.WithCurrentVersion(Version),
			)

			time.Sleep(checkDelay)
			versionChecker.CheckNewVersion(ctx)
		}()

		go helpers.GracefulShutdown(ctx, func(shutdownCtx context.Context) error {
			log.Println("shutdown signal received")

			return app.Shutdown(shutdownCtx)
		})

		app.Wait()
		output.Info("Server was stopped")
	} else {
		app := uncorsapp.NewUncorsApp(
			Version,
			fs,
			viperInstance,
			uncorsConfig,
			func() *config.UncorsConfig { return loadConfiguration(viperInstance, fs) },
		)
		p := tea.NewProgram(app)
		if _, err := p.Run(); err != nil {
			log.Fatal(err)
		}
	}

	return 0
}

func loadConfiguration(viperInstance *viper.Viper, fs afero.Fs) *config.UncorsConfig {
	uncorsConfig := config.LoadConfiguration(viperInstance, os.Args)

	err := validators.ValidateConfig(uncorsConfig, fs)
	if err != nil {
		panic(err)
	}

	if uncorsConfig.Debug {
		logFile, err := os.OpenFile("uncors.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModeAppend)
		if err != nil {
			panic(fmt.Sprintf("Failed to open log file: %v", err))
		}

		log.SetOutput(logFile)
		log.Print("Enabled debug messages")
	} else {
		log.SetOutput(io.Discard)
	}

	return uncorsConfig
}
