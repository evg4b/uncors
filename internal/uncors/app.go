package uncors

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"strconv"
	"sync"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/cache"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/internal/tui"

	"github.com/evg4b/uncors/internal/config"
	"github.com/spf13/afero"
)

const baseAddress = "127.0.0.1"

type Uncors struct {
	fs      afero.Fs
	version string
	output  contracts.Output
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

func CreateUncors(fs afero.Fs, output contracts.Output, version string) *Uncors {
	return &Uncors{
		fs:      fs,
		version: version,
		output:  output,
		server:  server.New(),
	}
}

func (app *Uncors) Start(ctx context.Context, uncorsConfig *config.UncorsConfig) error {
	tui.PrintLogo(app.output, app.version)
	app.output.Print("")
	app.output.WarnBox(tui.DisclaimerMessage)
	app.output.Print("")
	app.output.InfoBox(uncorsConfig.Mappings.String())
	app.output.Print("")

	targets, err := app.mappingsToTarget(uncorsConfig)
	if err != nil {
		return err
	}

	return app.server.Start(ctx, targets)
}

func (app *Uncors) Restart(ctx context.Context, uncorsConfig *config.UncorsConfig) error {
	app.output.Print("")
	app.output.Info("Restarting server....")
	app.output.Print("")

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

	app.output.Info(uncorsConfig.Mappings.String())
	app.output.Print("")

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
			tlsConfig, err = buildTLSConfig(app.fs, app.output, group.Mappings)
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
