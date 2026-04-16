package uncors

import (
	"context"
	"crypto/tls"
	"net"
	"os"
	"strconv"
	"sync"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/cache"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/internal/tui"

	"github.com/charmbracelet/log"
	"github.com/evg4b/uncors/internal/config"
	"github.com/spf13/afero"
)

const baseAddress = "127.0.0.1"

type Uncors struct {
	fs      afero.Fs
	version string
	logger  *log.Logger
	server  *server.Server

	cacheStorageOnce sync.Once
	cacheStorage     contracts.Cache
}

func CreateUncors(fs afero.Fs, logger *log.Logger, version string) *Uncors {
	return &Uncors{
		fs:      fs,
		version: version,
		logger:  logger,
		server:  server.New(),
	}
}

func (app *Uncors) Start(ctx context.Context, uncorsConfig *config.UncorsConfig) error {
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

func (app *Uncors) Restart(ctx context.Context, uncorsConfig *config.UncorsConfig) error {
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

func (app *Uncors) Close() error {
	return app.server.Close()
}

func (app *Uncors) Wait() {
	app.server.Wait()
}

func (app *Uncors) Shutdown(ctx context.Context) error {
	return app.server.Shutdown(ctx)
}

func (app *Uncors) getCacheStorage(cfg config.CacheConfig) contracts.Cache {
	app.cacheStorageOnce.Do(func() {
		app.cacheStorage = cache.NewRistrettoCache(cfg.MaxSize, cfg.ExpirationTime)
	})

	return app.cacheStorage
}

func (app *Uncors) mappingsToTarget(uncorsConfig *config.UncorsConfig) ([]server.Target, error) {
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
			TLSConfig: tlsConfig,
		})
	}

	return targets, nil
}
