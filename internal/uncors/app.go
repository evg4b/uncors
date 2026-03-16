package uncors

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"os"
	"strconv"
	"sync"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/cache"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/internal/tui"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/log"
	"github.com/spf13/afero"
)

const baseAddress = "127.0.0.1"

type Uncors struct {
	fs      afero.Fs
	version string
	logger  *log.Logger
	stdout  io.Writer
	server  *server.Server

	cacheStorageOnce sync.Once
	cacheStorage     contracts.Cache
	closers          []io.Closer
}

func (app *Uncors) registerCloser(c io.Closer) {
	app.closers = append(app.closers, c)
}

func (app *Uncors) closeAll() {
	for _, c := range app.closers {
		_ = c.Close()
	}

	app.closers = nil
}

func CreateUncors(fs afero.Fs, logger *log.Logger, version string) *Uncors {
	return &Uncors{
		fs:      fs,
		version: version,
		logger:  logger,
		stdout:  os.Stdout,
		server:  server.New(),
	}
}

func (app *Uncors) Start(ctx context.Context, uncorsConfig *config.UncorsConfig) error {
	tui.PrintLogo(app.stdout, app.version)
	app.logger.Print("")
	tui.PrintWarningBox(app.stdout, DisclaimerMessage)
	app.logger.Print("")
	tui.PrintInfoBox(app.stdout, uncorsConfig.Mappings.String())
	app.logger.Print("")

	targets, err := app.mappingsToTarget(uncorsConfig)
	if err != nil {
		return err
	}

	app.server.Start(ctx, targets)

	return nil
}

func (app *Uncors) Restart(ctx context.Context, uncorsConfig *config.UncorsConfig) error {
	app.logger.Print("")
	app.logger.Info("Restarting server....")
	app.logger.Print("")

	// Snapshot current closers so they can be drained after the new handlers
	// are running (new closers will be registered during mappingsToTarget).
	previous := app.closers
	app.closers = nil

	targets, err := app.mappingsToTarget(uncorsConfig)
	if err != nil {
		return err
	}

	err = app.server.Restart(ctx, targets)
	if err != nil {
		return err
	}

	// Flush and close the previous set of HAR writers now that new ones are live.
	for _, c := range previous {
		_ = c.Close()
	}

	app.logger.Info(uncorsConfig.Mappings.String())
	app.logger.Print("")

	return nil
}

func (app *Uncors) Close() error {
	app.closeAll()

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
			tlsConfig, err = buildTLSConfig(app.fs, app.logger, group.Mappings)
			if err != nil {
				return nil, err
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
