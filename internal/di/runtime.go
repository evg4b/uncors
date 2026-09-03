package di

import (
	"errors"
	"io"
	"net"
	"net/http"
	"slices"
	"strconv"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/cache"
	"github.com/evg4b/uncors/internal/handler/har"
	"github.com/evg4b/uncors/internal/handler/router"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/server"
)

const baseAddress = "127.0.0.1"

// Runtime owns everything derived from a single UncorsConfig: the response
// cache, the HAR writers, the routers built from the mappings and the server
// targets that serve them.
//
// Those resources have a configuration lifetime, not a process lifetime: a hot
// reload replaces all of them. Closing a Runtime releases exactly the resources
// that generation created, which is what keeps reloads from leaking goroutines
// and from leaving several HAR writers racing over the same file.
type Runtime struct {
	container   *Container
	cacheConfig *config.CacheConfig
	cacheStore  contracts.Cache

	targets []server.Target
	closers []io.Closer
}

// BuildRuntime materialises a configuration generation. The caller owns the
// result and must Close it once the generation stops being current.
func (c *Container) BuildRuntime(uncorsConfig *config.UncorsConfig) (*Runtime, error) {
	runtime := &Runtime{
		container:   c,
		cacheConfig: &uncorsConfig.CacheConfig,
	}

	targets, err := runtime.buildTargets(uncorsConfig)
	if err != nil {
		return nil, errors.Join(err, runtime.Close())
	}

	runtime.targets = targets

	return runtime, nil
}

// Targets returns the server targets this generation should be served on.
func (r *Runtime) Targets() []server.Target {
	return r.targets
}

// Close releases every resource created for this generation, in reverse
// creation order. It is safe to call more than once.
func (r *Runtime) Close() error {
	errs := make([]error, 0, len(r.closers))

	for _, closer := range slices.Backward(r.closers) {
		err := closer.Close()
		if err != nil {
			errs = append(errs, err)
		}
	}

	r.closers = nil
	r.cacheStore = nil

	return errors.Join(errs...)
}

// Cache lazily creates the response cache for this generation, so that a config
// without caching never allocates one and a reload always picks up the current
// cache-config.
func (r *Runtime) Cache() contracts.Cache {
	if r.cacheStore == nil {
		r.cacheStore = register(r, cache.NewRistrettoCache(r.cacheConfig.MaxSize, r.cacheConfig.ExpirationTime))
	}

	return r.cacheStore
}

func (r *Runtime) CacheMiddleware(globs config.CacheGlobs) contracts.Middleware {
	return infra.NewPrefixedMiddleware(
		cache.NewMiddleware(
			cache.WithMethods(r.cacheConfig.Methods),
			cache.WithCacheStorage(r.Cache()),
			cache.WithGlobs(globs),
		),
		"CACHE",
	)
}

func (r *Runtime) HARMiddleware(harConfig *config.HARConfig) contracts.Middleware {
	writer := register(r, har.NewWriter(harConfig.File))

	return har.NewMiddleware(
		har.WithWriter(writer),
		har.WithCaptureSecureHeaders(harConfig.CaptureSecureHeaders),
	)
}

func (r *Runtime) StaticMiddleware(path string, dir config.StaticDirectory) contracts.Middleware {
	return r.container.StaticMiddleware(path, dir)
}

func (r *Runtime) RewriteMiddleware(rewriting *config.RewritingOption) contracts.Middleware {
	return r.container.RewriteMiddleware(rewriting)
}

func (r *Runtime) OptionsMiddleware(cfg config.OptionsHandling) contracts.Middleware {
	return r.container.OptionsMiddleware(cfg)
}

func (r *Runtime) MockHandler(response *config.Response) contracts.Handler {
	return r.container.MockHandler(response)
}

func (r *Runtime) ScriptHandler(scriptConfig *config.Script) contracts.Handler {
	return r.container.ScriptHandler(scriptConfig)
}

func (r *Runtime) buildTargets(uncorsConfig *config.UncorsConfig) ([]server.Target, error) {
	groupedMappings := uncorsConfig.Mappings.GroupByPort()
	targets := make([]server.Target, 0, len(groupedMappings))
	errs := make([]error, 0, len(groupedMappings))

	for _, group := range groupedMappings {
		handler, err := r.router(group.Mappings, uncorsConfig.Proxy)
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

func (r *Runtime) router(mappings config.Mappings, proxyURL string) (contracts.Handler, error) {
	muxRouter, err := router.NewRouter(
		mappings,
		router.WithDiContainer(r),
		router.ForRouterWithDefaultHandler(r.container.ProxyHandler(mappings, r.httpClient(proxyURL))),
		router.ForRouterWithCacheMiddlewareFactory(r.CacheMiddleware),
	)

	return infra.CastToContractsHandler(muxRouter), err
}

// httpClient builds the upstream client for this generation and binds its
// connection pool to the generation's lifetime. Without this the idle
// connections of every superseded configuration survive until the process
// exits.
func (r *Runtime) httpClient(proxyURL string) *http.Client {
	client := infra.MakeHTTPClient(proxyURL)

	register(r, transportCloser{client: client})

	return client
}

// transportCloser releases a client's idle connections. http.Client has no
// Close, so this adapts the one thing that actually needs releasing.
type transportCloser struct {
	client *http.Client
}

func (t transportCloser) Close() error {
	t.client.CloseIdleConnections()

	return nil
}

// register binds a resource to the generation's lifetime.
func register[T io.Closer](runtime *Runtime, resource T) T {
	runtime.closers = append(runtime.closers, resource)

	return resource
}
