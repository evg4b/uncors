package uncors

import (
	"sync"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler"
	"github.com/evg4b/uncors/internal/handler/cache"
	"github.com/evg4b/uncors/internal/handler/mock"
	"github.com/evg4b/uncors/internal/handler/options"
	"github.com/evg4b/uncors/internal/handler/proxy"
	"github.com/evg4b/uncors/internal/handler/rewrite"
	"github.com/evg4b/uncors/internal/handler/script"
	"github.com/evg4b/uncors/internal/handler/static"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/spf13/afero"
)

func (app *Uncors) buildHandlerForMappings(
	uncorsConfig *config.UncorsConfig,
	mappings config.Mappings,
) *handler.RequestHandler {
	cacheConfig := uncorsConfig.CacheConfig

	// cacheStorage is shared across all mappings — created once on first cache hit.
	var (
		cacheStorageOnce sync.Once
		cacheStorage     contracts.Cache
	)
	getCacheStorage := func() contracts.Cache {
		cacheStorageOnce.Do(func() {
			cacheStorage = cache.NewRistrettoCache(cacheConfig.MaxSize, cacheConfig.ExpirationTime)
		})

		return cacheStorage
	}

	return handler.NewUncorsRequestHandler(
		handler.WithMappings(mappings),
		handler.WithProxyHandler(contracts.LazyHandler(func() contracts.Handler {
			return proxy.NewProxyHandler(
				proxy.WithURLReplacerFactory(urlreplacer.NewURLReplacerFactory(mappings)),
				proxy.WithHTTPClient(infra.MakeHTTPClient(uncorsConfig.Proxy)),
				proxy.WithProxyLogger(NewProxyLogger(app.logger)),
				proxy.WithRewriteLogger(NewRewriteLogger(app.logger)),
			)
		})),
		handler.WithCacheMiddlewareFactory(func(globs config.CacheGlobs) contracts.Middleware {
			return contracts.LazyMiddleware(func() contracts.Middleware {
				return cache.NewMiddleware(
					cache.WithLogger(NewCacheLogger(app.logger)),
					cache.WithMethods(cacheConfig.Methods),
					cache.WithCacheStorage(getCacheStorage()),
					cache.WithGlobs(globs),
				)
			})
		}),
		handler.WithOptionsHandlerFactory(func(cfg config.OptionsHandling) contracts.Middleware {
			return contracts.LazyMiddleware(func() contracts.Middleware {
				return options.NewMiddleware(
					options.WithLogger(NewOptionsLogger(app.logger)),
					options.WithHeaders(cfg.Headers),
					options.WithCode(cfg.Code),
				)
			})
		}),
		handler.WithStaticHandlerFactory(func(path string, dir config.StaticDirectory) contracts.Middleware {
			return contracts.LazyMiddleware(func() contracts.Middleware {
				return static.NewStaticMiddleware(
					static.WithFileSystem(afero.NewBasePathFs(app.fs, dir.Dir)),
					static.WithIndex(dir.Index),
					static.WithLogger(NewStaticLogger(app.logger)),
					static.WithPrefix(path),
				)
			})
		}),
		handler.WithMockHandlerFactory(func(response config.Response) contracts.Handler {
			return contracts.LazyHandler(func() contracts.Handler {
				return mock.NewMockHandler(
					mock.WithLogger(NewMockLogger(app.logger)),
					mock.WithResponse(response),
					mock.WithFileSystem(app.fs),
					mock.WithAfter(time.After),
				)
			})
		}),
		handler.WithScriptHandlerFactory(func(s config.Script) contracts.Handler {
			return contracts.LazyHandler(func() contracts.Handler {
				return script.NewHandler(
					script.WithLogger(NewScriptLogger(app.logger)),
					script.WithScript(s),
					script.WithFileSystem(app.fs),
				)
			})
		}),
		handler.WithRewriteHandlerFactory(func(rewriting config.RewritingOption) contracts.Middleware {
			return contracts.LazyMiddleware(func() contracts.Middleware {
				return rewrite.NewMiddleware(rewrite.WithRewritingOptions(rewriting))
			})
		}),
	)
}
