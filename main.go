package main

import (
	"fmt"
	"os"

	"github.com/evg4b/uncors/internal/tui/request_tracker"
	"github.com/evg4b/uncors/internal/tui/styles"
	"github.com/muesli/termenv"

	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/version"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/charmbracelet/log"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/evg4b/uncors/internal/uncors"
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

	logPrinter := tui.NewPrinter()
	defer func() {
		if err := logPrinter.Close(); err != nil {
			panic(err)
		}
	}()

	log.SetOutput(logPrinter)
	log.SetReportTimestamp(false)
	log.SetReportCaller(false)
	log.SetStyles(styles.DefaultStyles())
	log.SetColorProfile(termenv.ColorProfile())

	log.Debugf("Starting Uncors %s", Version)

	pflag.Usage = func() {
		fmt.Print(uncors.Logo(Version)) //nolint:forbidigo
		helpers.FPrintf(os.Stdout, "Usage of %s:\n", os.Args[0])
		pflag.PrintDefaults()
	}

	fs := afero.NewOsFs()
	viperInstance := viper.GetViper()
	loader := tui.NewConfigLoader(viperInstance, fs)
	uncorsConfig := loader.Load()

	tracker := request_tracker.NewRequestTracker()

	ctx := context.Background()

	go version.CheckNewVersion(ctx, infra.MakeHTTPClient(uncorsConfig.Proxy), Version)

	model := uncors.NewUncorsModel(
		uncors.WithVersion(Version),
		uncors.WithLogPrinter(logPrinter),
		uncors.WithConfig(uncorsConfig),
		uncors.WithRequestTracker(tracker),
		uncors.WithConfigLoader(loader),
	)

	program := tea.NewProgram(model, tea.WithContext(ctx))
	if _, err := program.Run(); err != nil {
		log.Fatal(err)
	}
}
