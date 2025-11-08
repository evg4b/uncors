package uncors

import (
	"crypto/tls"
	"net"
	"os"
	"strconv"

	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/internal/tui"

	"github.com/charmbracelet/log"
	"github.com/evg4b/uncors/internal/config"
	"github.com/spf13/afero"
	"golang.org/x/net/context"
)

const baseAddress = "127.0.0.1"

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

func (app *App) Start(ctx context.Context, uncorsConfig *config.UncorsConfig) error {
	tui.PrintLogo(os.Stdout, app.version)
	log.Print("")
	tui.PrintWarningBox(os.Stdout, DisclaimerMessage)
	log.Print("")
	tui.PrintInfoBox(os.Stdout, uncorsConfig.Mappings.String())
	log.Print("")

	targets, err := app.mappingsToTarget(uncorsConfig)
	if err != nil {
		return err
	}

	app.server.Start(ctx, targets)

	return nil
}

func (app *App) Restart(ctx context.Context, uncorsConfig *config.UncorsConfig) error {
	log.Print("")
	log.Info("Restarting server....")
	log.Print("")

	targets, err := app.mappingsToTarget(uncorsConfig)
	if err != nil {
		return err
	}

	err = app.server.Restart(ctx, targets)
	if err != nil {
		return err
	}

	log.Info(uncorsConfig.Mappings.String())
	log.Print("")

	return nil
}

func (app *App) Close() error {
	return app.server.Close()
}

func (app *App) Wait() {
	app.server.Wait()
}

func (app *App) Shutdown(ctx context.Context) error {
	return app.server.Shutdown(ctx)
}

func (app *App) mappingsToTarget(uncorsConfig *config.UncorsConfig) ([]server.Target, error) {
	groupedMappings := uncorsConfig.Mappings.GroupByPort()

	targets := make([]server.Target, 0, len(groupedMappings))

	for _, group := range groupedMappings {
		var (
			tlsConfig *tls.Config
			err       error
		)

		if group.Scheme == "https" {
			tlsConfig, err = buildTLSConfig(app.fs, group.Mappings)
			if err != nil {
				return []server.Target{}, err
			}
		}

		targets = append(targets, server.Target{
			Address:   net.JoinHostPort(baseAddress, strconv.Itoa(group.Port)),
			Handler:   app.buildHandlerForMappings(uncorsConfig, group.Mappings),
			TLSConfgi: tlsConfig,
		})
	}

	return targets, nil
}
