package uncors

import (
	"os"

	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/samber/lo"

	"github.com/charmbracelet/log"
	"github.com/evg4b/uncors/internal/config"
	"github.com/spf13/afero"
	"golang.org/x/net/context"
)

type App struct {
	fs      afero.Fs
	version string
	cache   appCache
	logger  *log.Logger
	server  *server.Server
}

func CreateApp(fs afero.Fs, logger *log.Logger, version string) *App {
	return &App{
		fs:      fs,
		version: version,
		logger:  logger,
		server:  server.New(),
	}
}

func (app *App) Start(ctx context.Context, uncorsConfig *config.UncorsConfig) {
	tui.PrintLogo(os.Stdout, app.version)
	log.Print("")
	tui.PrintWarningBox(os.Stdout, DisclaimerMessage)
	log.Print("")
	tui.PrintInfoBox(os.Stdout, uncorsConfig.Mappings.String())
	log.Print("")

	targets := app.mappingsToTarget(uncorsConfig)

	app.server.Start(ctx, targets)
}

func (app *App) Restart(ctx context.Context, uncorsConfig *config.UncorsConfig) {
	log.Print("")
	log.Info("Restarting server....")
	log.Print("")

	err := app.server.Restart(ctx, app.mappingsToTarget(uncorsConfig))
	if err != nil {
		// TODO(v2.0): Replace panic with graceful error handling and user notification
		panic(err)
	}

	log.Info(uncorsConfig.Mappings.String())
	log.Print("")
}

func (app *App) Close() error {
	return nil
}

func (app *App) Wait() {
	app.server.Wait()
}

func (app *App) Shutdown(ctx context.Context) error {
	return app.server.Shutdown(ctx)
}

func (app *App) mappingsToTarget(uncorsConfig *config.UncorsConfig) []server.Target {
	return lo.Map(uncorsConfig.Mappings.GroupByPort(), func(group config.PortGroup, _ int) server.Target {
		return server.Target{
			Port:    group.Port,
			Handler: app.buildHandlerForMappings(uncorsConfig, uncorsConfig.Mappings),
		}
	})
}
