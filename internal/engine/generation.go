package engine

import (
	"context"
	"errors"
	"io"
	"net"
	"slices"
	"strconv"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/cache"
	"github.com/evg4b/uncors/internal/handler/har"
	"github.com/evg4b/uncors/internal/handler/mock"
	"github.com/evg4b/uncors/internal/handler/options"
	"github.com/evg4b/uncors/internal/handler/proxy"
	"github.com/evg4b/uncors/internal/handler/rewrite"
	"github.com/evg4b/uncors/internal/handler/router"
	"github.com/evg4b/uncors/internal/handler/script"
	"github.com/evg4b/uncors/internal/handler/static"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/internal/telemetry"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/spf13/afero"
	"go.opentelemetry.io/otel/attribute"
)

const baseAddress = "127.0.0.1"

type Generation struct {
	engine      *Engine
	cacheConfig *config.CacheConfig
	cacheStore  contracts.Cache

	targets []server.Target
	closers []io.Closer
}

var _ router.Factory = (*Generation)(nil)

func (e *Engine) BuildGeneration(uncorsConfig *config.UncorsConfig) (*Generation, error) {
	return e.buildGeneration(context.Background(), uncorsConfig)
}

func (e *Engine) buildGeneration(ctx context.Context, uncorsConfig *config.UncorsConfig) (*Generation, error) {
	_, span := telemetry.Start(ctx, "engine.build_generation")
	defer span.End()

	span.SetAttributes(attribute.Int("uncors.mappings", len(uncorsConfig.Mappings)))

	generation := &Generation{
		engine:      e,
		cacheConfig: &uncorsConfig.CacheConfig,
	}

	targets, err := generation.buildTargets(uncorsConfig)
	if err != nil {
		err = errors.Join(err, generation.Close())
		telemetry.Fail(span, err)

		return nil, err
	}

	generation.targets = targets

	return generation, nil
}

func (g *Generation) Targets() []server.Target {
	return g.targets
}

func (g *Generation) Close() error {
	errs := make([]error, 0, len(g.closers))

	for _, closer := range slices.Backward(g.closers) {
		err := closer.Close()
		if err != nil {
			errs = append(errs, err)
		}
	}

	g.closers = nil
	g.cacheStore = nil

	return errors.Join(errs...)
}

func (g *Generation) Cache() contracts.Cache {
	if g.cacheStore == nil {
		g.cacheStore = register(g, cache.NewRistrettoCache(g.cacheConfig.MaxSize, g.cacheConfig.ExpirationTime))
	}

	return g.cacheStore
}

func (g *Generation) CacheMiddleware(globs config.CacheGlobs) contracts.Middleware {
	return cache.NewMiddleware(
		cache.WithMethods(g.cacheConfig.Methods),
		cache.WithCacheStorage(g.Cache()),
		cache.WithGlobs(globs),
	)
}

func (g *Generation) HARMiddleware(harConfig *config.HARConfig) contracts.Middleware {
	writer := register(g, har.NewWriter(harConfig.File))

	return har.NewMiddleware(
		har.WithWriter(writer),
		har.WithCaptureSecureHeaders(harConfig.CaptureSecureHeaders),
	)
}

func (g *Generation) StaticMiddleware(path string, dir config.StaticDirectory) contracts.Middleware {
	return static.NewStaticMiddleware(
		static.WithFileSystem(afero.NewBasePathFs(g.engine.fs, dir.Dir)),
		static.WithIndex(dir.Index),
		static.WithPrefix(path),
	)
}

func (g *Generation) RewriteMiddleware(rewriting *config.RewritingOption) contracts.Middleware {
	return rewrite.NewMiddleware(rewrite.WithRewritingOptions(rewriting))
}

func (g *Generation) OptionsMiddleware(cfg config.OptionsHandling) contracts.Middleware {
	return options.NewMiddleware(
		options.WithHeaders(cfg.Headers),
		options.WithCode(cfg.Code),
	)
}

func (g *Generation) MockHandler(response *config.Response) contracts.Handler {
	return mock.NewMockHandler(
		mock.WithResponse(response),
		mock.WithFileSystem(g.engine.fs),
		mock.WithAfter(time.After),
	)
}

func (g *Generation) ScriptHandler(scriptConfig *config.Script) contracts.Handler {
	return script.NewHandler(
		script.WithScript(scriptConfig),
		script.WithFileSystem(g.engine.fs),
	)
}

func (g *Generation) ProxyHandler(mappings config.Mappings, proxyURL string) contracts.Handler {
	return proxy.NewProxyHandler(
		proxy.WithURLReplacerFactory(urlreplacer.NewURLReplacerFactory(mappings)),
		proxy.WithHTTPClient(infra.MakeHTTPClient(proxyURL)),
	)
}

func (g *Generation) buildTargets(uncorsConfig *config.UncorsConfig) ([]server.Target, error) {
	groupedMappings := uncorsConfig.Mappings.GroupByPort()
	targets := make([]server.Target, 0, len(groupedMappings))
	errs := make([]error, 0, len(groupedMappings))

	for _, group := range groupedMappings {
		handler, err := g.router(group.Mappings, uncorsConfig.Proxy)
		if err != nil {
			errs = append(errs, err)

			continue
		}

		targets = append(targets, server.Target{
			Address:   net.JoinHostPort(baseAddress, strconv.Itoa(group.Port)),
			Handler:   handler,
			EnableTLS: group.Scheme == "https",
		})
	}

	return targets, errors.Join(errs...)
}

func (g *Generation) router(mappings config.Mappings, proxyURL string) (contracts.Handler, error) {
	muxRouter, err := router.NewRouter(
		mappings,
		router.WithFactory(g),
		router.ForRouterWithDefaultHandler(g.ProxyHandler(mappings, proxyURL)),
		router.ForRouterWithCacheMiddlewareFactory(g.CacheMiddleware),
	)

	return infra.CastToContractsHandler(muxRouter), err
}

func register[T io.Closer](generation *Generation, resource T) T {
	generation.closers = append(generation.closers, resource)

	return resource
}
