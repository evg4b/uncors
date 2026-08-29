package di

import (
	"errors"
	"io"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/cache"
	"github.com/evg4b/uncors/internal/handler/har"
	"github.com/evg4b/uncors/internal/handler/router"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/internal/tui/styles"
)

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
		ttl := time.Duration(r.cacheConfig.ExpirationTime)
		r.cacheStore = register(r, cache.NewRistrettoCache(r.cacheConfig.MaxSize, ttl))
	}

	return r.cacheStore
}

func (r *Runtime) cacheMiddleware(globs config.CacheGlobs) contracts.Middleware {
	return infra.NewPrefixedMiddleware(
		cache.NewMiddleware(
			cache.WithMethods(r.cacheConfig.Methods),
			cache.WithCacheStorage(r.Cache()),
			cache.WithGlobs(globs),
		).Wrap,
		styles.CacheStyle.Render("CACHE"),
	)
}

// harMiddleware records this generation's traffic. The writer itself is process
// scoped and keyed by path, so a reload keeps appending to the same archive
// instead of starting a second writer over it.
func (r *Runtime) harMiddleware(harConfig *config.HARConfig) contracts.Middleware {
	return har.NewMiddleware(
		har.WithWriter(r.container.harWriters().For(harConfig.File)),
		har.WithCaptureSecureHeaders(harConfig.CaptureSecureHeaders),
	).Wrap
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
			Address:     net.JoinHostPort(uncorsConfig.Listen, strconv.Itoa(group.Port)),
			Handler:     handler,
			EnableTLS:   group.Scheme == "https",
			DefaultHost: defaultHostOf(group.Mappings),
		})
	}

	return targets, errors.Join(errs...)
}

func (r *Runtime) router(mappings config.Mappings, proxyURL string) (http.Handler, error) {
	proxyHandler, err := r.container.ProxyHandler(mappings, proxyURL)
	if err != nil {
		return nil, err
	}

	muxRouter, err := router.NewRouter(mappings, router.Deps{
		Proxy:   proxyHandler,
		Static:  r.container.StaticMiddleware,
		Rewrite: r.container.RewriteMiddleware,
		Options: r.container.OptionsMiddleware,
		Mock:    r.container.MockHandler,
		Script:  r.container.ScriptHandler,
		HAR:     r.harMiddleware,
		Cache:   r.cacheMiddleware,
	})

	return muxRouter, err
}

// defaultHostOf picks the certificate host to use for a client that sends no
// SNI. A wildcard bind has no local address to fall back on, so the mapping's
// own hostname is the only sensible answer.
func defaultHostOf(mappings config.Mappings) string {
	for _, mapping := range mappings {
		hostname := mapping.From.Hostname
		if hostname != "" && !strings.ContainsAny(hostname, "{}") {
			return hostname
		}
	}

	return ""
}

// register binds a resource to the generation's lifetime.
func register[T io.Closer](runtime *Runtime, resource T) T {
	runtime.closers = append(runtime.closers, resource)

	return resource
}
