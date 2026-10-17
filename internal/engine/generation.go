package engine

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/events"
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
	fs          afero.Fs
	publisher   events.Publisher
	cacheConfig *config.CacheConfig
	cacheStore  contracts.Cache
	proxyURL    string
	client      *http.Client

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
		fs:          e.fs,
		publisher:   e.publisher,
		cacheConfig: &uncorsConfig.CacheConfig,
		proxyURL:    uncorsConfig.Proxy,
	}

	targets, err := generation.build(uncorsConfig)
	if err != nil {
		telemetry.Fail(span, err)

		return nil, errors.Join(err, generation.Close())
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
	g.client = nil

	return errors.Join(errs...)
}

func (g *Generation) Cache() contracts.Cache {
	if g.cacheStore == nil {
		_ = g.openCache()
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
	writer := register(g, har.NewWriter(harConfig.File,
		har.WithFileSystem(g.fs),
		har.WithPublisher(g.publisher),
	))

	return har.NewMiddleware(
		har.WithWriter(writer),
		har.WithCaptureSecureHeaders(harConfig.CaptureSecureHeaders),
	)
}

func (g *Generation) StaticMiddleware(path string, dir config.StaticDirectory) contracts.Middleware {
	return static.NewStaticMiddleware(
		static.WithFileSystem(afero.NewBasePathFs(g.fs, dir.Dir)),
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
		mock.WithFileSystem(g.fs),
		mock.WithAfter(time.After),
	)
}

func (g *Generation) ScriptHandler(scriptConfig *config.Script) contracts.Handler {
	return script.NewHandler(
		script.WithScript(scriptConfig),
		script.WithFileSystem(g.fs),
	)
}

func (g *Generation) ProxyHandler(replacers urlreplacer.ReplacerFactory) contracts.Handler {
	return proxy.NewProxyHandler(
		proxy.WithURLReplacerFactory(replacers),
		proxy.WithHTTPClient(g.client),
	)
}

func (g *Generation) openCache() error {
	store, err := cache.NewRistrettoCache(g.cacheConfig.MaxSize, g.cacheConfig.ExpirationTime)
	if err != nil {
		return err
	}

	g.cacheStore = register(g, store)

	return nil
}

// Shared by every listener of the generation; closing it drops idle upstream connections.
func (g *Generation) openClient() error {
	client, err := infra.MakeHTTPClient(g.proxyURL)
	if err != nil {
		return err
	}

	g.client = client
	register(g, closerFunc(func() error {
		client.CloseIdleConnections()

		return nil
	}))

	return nil
}

func (g *Generation) build(uncorsConfig *config.UncorsConfig) ([]server.Target, error) {
	err := g.openClient()
	if err != nil {
		return nil, err
	}

	if slices.ContainsFunc(uncorsConfig.Mappings, func(mapping config.Mapping) bool {
		return len(mapping.Cache) > 0
	}) {
		err = g.openCache()
		if err != nil {
			return nil, err
		}
	}

	groupedMappings, err := uncorsConfig.Mappings.GroupByPort()
	if err != nil {
		return nil, err
	}

	targets := make([]server.Target, 0, len(groupedMappings))

	for _, group := range groupedMappings {
		replacers, err := urlreplacer.NewURLReplacerFactory(group.Mappings)
		if err != nil {
			return nil, err
		}

		targets = append(targets, server.Target{
			Address: net.JoinHostPort(baseAddress, strconv.Itoa(group.Port)),
			Handler: router.NewRouter(
				group.Mappings,
				router.WithFactory(g),
				router.WithDefaultHandler(g.ProxyHandler(replacers)),
				router.WithCacheMiddlewareFactory(g.CacheMiddleware),
			),
			EnableTLS: group.Scheme == "https",
		})
	}

	return targets, nil
}

type closerFunc func() error

func (f closerFunc) Close() error {
	return f()
}

func register[T io.Closer](generation *Generation, resource T) T {
	generation.closers = append(generation.closers, resource)

	return resource
}
