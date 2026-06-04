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
	"github.com/samber/lo"

	"github.com/evg4b/uncors/internal/config"
	"github.com/spf13/afero"
)

const baseAddress = "127.0.0.1"

type Uncors struct {
	fs      afero.Fs
	version string
	output  contracts.Output
	Server  *server.Server

	cacheStorage contracts.Cache
	closers      []io.Closer
}

func CreateUncors(fs afero.Fs, output contracts.Output, version string) *Uncors {
	return &Uncors{
		fs:      fs,
		version: version,
		output:  output,
		Server:  server.New(server.NewHostCertManager(fs)),
	}
}

func (app *Uncors) WithTracker(tracker *server.RequestTracker) *Uncors {
	app.Server.Tracker = tracker

	return app
}

func (app *Uncors) Start(ctx context.Context, uncorsConfig *config.UncorsConfig) error {
	tui.PrintLogo(app.output, app.version)
	app.output.Print("")
	app.output.WarnBox(tui.DisclaimerMessage)
	app.output.Print("")
	app.output.InfoBox(uncorsConfig.Mappings.String())
	app.output.Print("")

	targets := app.mappingsToTarget(uncorsConfig)

	return app.Server.Start(ctx, targets)
}

func (app *Uncors) Restart(ctx context.Context, uncorsConfig *config.UncorsConfig) error {
	app.output.Info("Restarting server....")

	previous := app.closers
	app.closers = nil
	// Drop the cache so the reloaded cache-config (size/TTL) takes effect; the
	// old instance is in `previous` and is closed below.
	app.cacheStorage = nil

	targets := app.mappingsToTarget(uncorsConfig)

	err := app.Server.Restart(ctx, targets)
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

	return app.Server.Close()
}

func (app *Uncors) Wait() {
	app.Server.Wait()
}

func (app *Uncors) Shutdown(ctx context.Context) error {
	return app.Server.Shutdown(ctx)
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

func (app *Uncors) mappingsToTarget(uncorsConfig *config.UncorsConfig) []server.Target {
	return lo.Map(uncorsConfig.Mappings.GroupByPort(), func(group config.PortGroup, _ int) server.Target {
		handler := handler.NewUncorsRequestHandler(
			handler.WithMappings(group.Mappings),
			handler.WithProxyHandler(app.buildProxyHandler(uncorsConfig, group.Mappings)),
			handler.WithCacheMiddlewareFactory(app.buildCacheMiddlewareFactory(uncorsConfig.CacheConfig)),
			handler.WithOptionsHandlerFactory(app.buildOptionsMiddlewareFactory()),
			handler.WithStaticHandlerFactory(app.buildStaticMiddlewareFactory()),
			handler.WithMockHandlerFactory(app.buildMockHandlerFactory()),
			handler.WithScriptHandlerFactory(app.buildScriptHandlerFactory()),
			handler.WithRewriteHandlerFactory(app.buildRewriteMiddlewareFactory()),
			handler.WithOutput(app.output),
			handler.WithHARMiddlewareFactory(app.buildHARMiddlewareFactory()),
		)
		trackedHandler := app.Server.Tracker.Wrap(handler)

		return server.Target{
			Address:   net.JoinHostPort(baseAddress, strconv.Itoa(group.Port)),
			Handler:   trackedHandler,
			EnableTLS: group.Scheme == "https",
		}
	})
}
