package uncors

import (
	"context"
	"errors"
	"io"
	"net"
	"strconv"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/di"
	"github.com/evg4b/uncors/internal/handler"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/internal/tui"

	"github.com/evg4b/uncors/internal/config"
	"github.com/spf13/afero"
)

const baseAddress = "127.0.0.1"

type Uncors struct {
	fs afero.Fs

	output    contracts.Output
	server    *server.Server
	container *di.Container

	closers []io.Closer
}

func CreateUncors(container *di.Container) *Uncors {
	return &Uncors{
		fs:        container.Fs(),
		output:    container.CliOutput(),
		container: container,
		server:    container.Server(),
	}
}

func (app *Uncors) Start(ctx context.Context, uncorsConfig *config.UncorsConfig) error {
	tui.PrintLogo(app.output, app.container.Version())
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
	return errors.Join(
		app.closeAll(),
		app.server.Close(),
	)
}

func (app *Uncors) Wait() {
	app.server.Wait()
}

func (app *Uncors) Shutdown(ctx context.Context) error {
	return app.server.Shutdown(ctx)
}

func (app *Uncors) registerCloser(c io.Closer) {
	app.closers = append(app.closers, c)
}

func (app *Uncors) closeAll() error {
	var errs []error

	for _, c := range app.closers {
		err := c.Close()
		if err != nil {
			errs = append(errs, err)
		}
	}

	app.closers = nil

	return errors.Join(errs...)
}

func (app *Uncors) mappingsToTarget(uncorsConfig *config.UncorsConfig) ([]server.Target, error) {
	targets := make([]server.Target, 0, len(uncorsConfig.Mappings.GroupByPort()))

	for _, group := range uncorsConfig.Mappings.GroupByPort() {
		router, err := handler.NewRouter(
			group.Mappings,
			handler.WithDiContainer(app.container),
			handler.ForRouterWithDefaultHandler(app.buildProxyHandler(uncorsConfig.Proxy, group.Mappings)),
			handler.ForRouterWithCacheMiddlewareFactory(app.buildCacheMiddlewareFactory(&uncorsConfig.CacheConfig)),
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
			Handler:   infra.CastToContractsHandler(router),
			EnableTLS: group.Scheme == "https",
		})
	}

	return targets, nil
}
