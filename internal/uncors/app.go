package uncors

import (
	"context"
	"io"
	"net"
	"strconv"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler"
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

	cacheStorage contracts.Cache
	closers      []io.Closer
}

func CreateUncors(fs afero.Fs, tracker *server.RequestTracker, output contracts.Output, version string) *Uncors {
	return &Uncors{
		fs:      fs,
		version: version,
		output:  output,
		server:  server.New(server.NewHostCertManager(fs), tracker),
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
	app.output.Info("Restarting server....")

	previous := app.closers
	app.closers = nil
	// Drop the cache so the reloaded cache-config (size/TTL) takes effect; the
	// old instance is in `previous` and is closed below.
	app.cacheStorage = nil

	targets, err := app.mappingsToTarget(uncorsConfig)
	if err != nil {
		return err
	}

	err = app.server.Restart(ctx, targets)
	if err != nil {
		return err
	}

	for _, c := range previous {
		_ = c.Close()
	}

	app.output.InfoBox(
		"Server restarted",
		uncorsConfig.Mappings.String(),
	)

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

// getCacheStorage lazily builds a single cache shared by every cache-enabled
// mapping in the current build. It is called only during Start/Restart (a single
// goroutine), so the nil check needs no synchronisation. The cache is registered
// as a closer so the previous instance is released on the next Restart, which is
// also what lets a changed cache-config take effect on reload.
func (app *Uncors) getCacheStorage(cfg config.CacheConfig) contracts.Cache {
	if app.cacheStorage == nil {
		storage := cache.NewRistrettoCache(cfg.MaxSize, cfg.ExpirationTime)
		app.cacheStorage = storage
		app.registerCloser(storage)
	}

	return app.cacheStorage
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

func (app *Uncors) mappingsToTarget(uncorsConfig *config.UncorsConfig) ([]server.Target, error) {
	targets := make([]server.Target, 0, len(uncorsConfig.Mappings.GroupByPort()))

	for _, group := range uncorsConfig.Mappings.GroupByPort() {
		router, err := handler.NewRouter(
			group.Mappings,
			handler.ForRouterWithDefaultHandler(app.buildProxyHandler(uncorsConfig, group.Mappings)),
			handler.ForRouterWithCacheMiddlewareFactory(app.buildCacheMiddlewareFactory(uncorsConfig.CacheConfig)),
			handler.ForRouterWithOptionsMiddlewareFactory(app.buildOptionsMiddlewareFactory()),
			handler.ForRouterWithStaticMiddlewareFactory(app.buildStaticMiddlewareFactory()),
			handler.ForRouterWithMockHandlerFactory(app.buildMockHandlerFactory()),
			handler.ForRouterWithScriptHandlerFactory(app.buildScriptHandlerFactory()),
			handler.ForRouterWithRewriteMiddlewareFactory(app.buildRewriteMiddlewareFactory()),
			handler.ForRouterWithHARMiddlewareFactory(app.buildHARMiddlewareFactory()),
		)
		if err != nil {
			return nil, err
		}

		targets = append(targets, server.Target{
			Address:   net.JoinHostPort(baseAddress, strconv.Itoa(group.Port)),
			Handler:   contracts.CastToContractsHandler(router),
			EnableTLS: group.Scheme == "https",
		})
	}

	return targets, nil
}
