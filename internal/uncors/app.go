package uncors

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
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

	tracker *server.RequestTracker

	cacheStorageOnce sync.Once
	cacheStorage     contracts.Cache
}

func CreateUncors(fs afero.Fs, output contracts.Output, version string) *Uncors {
	return &Uncors{
		fs:      fs,
		version: version,
		output:  output,
		server:  server.New(),
	}
}

func (app *Uncors) WithTracker(tracker *server.RequestTracker) *Uncors {
	app.tracker = tracker

	return app
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
	targets, err := app.mappingsToTarget(uncorsConfig)
	if err != nil {
		return err
	}

	err = app.server.Restart(ctx, targets)
	if err != nil {
		return err
	}

	app.output.InfoBox(
		"Server restarted",
		uncorsConfig.Mappings.String(),
	)

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
			tlsConfig, err = buildTLSConfig(app.fs, app.output, group.Mappings)
			if err != nil {
				return nil, err
			}
		}

		innerHandler := contracts.Handler(app.buildHandlerForMappings(uncorsConfig, group.Mappings))

		var httpHandler http.Handler
		if app.tracker != nil {
			httpHandler = app.tracker.Wrap(innerHandler)
		} else {
			httpHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				innerHandler.ServeHTTP(contracts.WrapResponseWriter(w), r)
			})
		}

		targets = append(targets, server.Target{
			Address:   net.JoinHostPort(baseAddress, strconv.Itoa(group.Port)),
			Handler:   httpHandler,
			TLSConfig: tlsConfig,
		})
	}

	return targets, nil
}
