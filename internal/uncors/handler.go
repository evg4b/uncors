package uncors

import (
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler"
	cache2 "github.com/evg4b/uncors/internal/handler/cache"
	"github.com/evg4b/uncors/internal/handler/mock"
	"github.com/evg4b/uncors/internal/handler/options"
	"github.com/evg4b/uncors/internal/handler/proxy"
	"github.com/evg4b/uncors/internal/handler/rewrite"
	"github.com/evg4b/uncors/internal/handler/script"
	"github.com/evg4b/uncors/internal/handler/static"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/patrickmn/go-cache"
	"github.com/spf13/afero"
)

type appCache struct {
	staticHandlerFactory handler.RequestHandlerOption
	mockHandlerFactory   handler.RequestHandlerOption
	scriptHandlerFactory handler.RequestHandlerOption
}

func (app *App) buildHandlerForMappings(
	uncorsConfig *config.UncorsConfig,
	mappings config.Mappings,
) *handler.RequestHandler {
	portHandler := handler.NewUncorsRequestHandler(
		handler.WithMappings(mappings),
		handler.WithLogger(NewMockLogger(app.logger)),
		handler.WithCacheMiddlewareFactory(func(globs config.CacheGlobs) contracts.Middleware {
			cacheConfig := uncorsConfig.CacheConfig
			// TODO: Add cache storage reusage
			cacheStorage := cache.New(cacheConfig.ExpirationTime, cacheConfig.ClearTime)

			return cache2.NewMiddleware(
				cache2.WithLogger(NewCacheLogger(app.logger)),
				cache2.WithMethods(cacheConfig.Methods),
				cache2.WithCacheStorage(cacheStorage),
				cache2.WithGlobs(globs),
			)
		}),
		handler.WithRewriteHandlerFactory(func(rewriting config.RewritingOption) contracts.Middleware {
			return rewrite.NewMiddleware(rewrite.WithRewritingOptions(rewriting))
		}),
		handler.WithOptionsHandlerFactory(func(config config.OptionsHandling) contracts.Middleware {
			return options.NewMiddleware(
				options.WithLogger(NewOptionsLogger(app.logger)),
				options.WithHeaders(config.Headers),
				options.WithCode(config.Code),
			)
		}),
		handler.WithProxyHandlerFactory(func() contracts.Handler {
			factory := urlreplacer.NewURLReplacerFactory(mappings)
			httpClient := infra.MakeHTTPClient(uncorsConfig.Proxy)

			return proxy.NewProxyHandler(
				proxy.WithURLReplacerFactory(factory),
				proxy.WithHTTPClient(httpClient),
				proxy.WithProxyLogger(NewProxyLogger(app.logger)),
				proxy.WithRewriteLogger(NewRewriteLogger(app.logger)),
			)
		}),
		app.getWithStaticHandlerFactory(),
		app.getMockHandlerFactory(),
		app.getScriptHandlerFactory(),
	)

	return portHandler
}

func (app *App) getMockHandlerFactory() handler.RequestHandlerOption {
	if app.cache.mockHandlerFactory == nil {
		factoryFunc := func(response config.Response) contracts.Handler {
			return mock.NewMockHandler(
				mock.WithLogger(NewMockLogger(app.logger)),
				mock.WithResponse(response),
				mock.WithFileSystem(app.fs),
				mock.WithAfter(time.After),
			)
		}
		app.cache.mockHandlerFactory = handler.WithMockHandlerFactory(factoryFunc)
	}

	return app.cache.mockHandlerFactory
}

func (app *App) getWithStaticHandlerFactory() handler.RequestHandlerOption {
	if app.cache.staticHandlerFactory == nil {
		factoryFunc := func(path string, dir config.StaticDirectory) contracts.Middleware {
			return static.NewStaticMiddleware(
				static.WithFileSystem(afero.NewBasePathFs(app.fs, dir.Dir)),
				static.WithIndex(dir.Index),
				static.WithLogger(NewStaticLogger(app.logger)),
				static.WithPrefix(path),
			)
		}

		app.cache.staticHandlerFactory = handler.WithStaticHandlerFactory(factoryFunc)
	}

	return app.cache.staticHandlerFactory
}

func (app *App) getScriptHandlerFactory() handler.RequestHandlerOption {
	if app.cache.scriptHandlerFactory == nil {
		factoryFunc := func(s config.Script) contracts.Handler {
			return script.NewHandler(
				script.WithLogger(NewScriptLogger(app.logger)),
				script.WithScript(s),
				script.WithFileSystem(app.fs),
			)
		}
		app.cache.scriptHandlerFactory = handler.WithScriptHandlerFactory(factoryFunc)
	}

	return app.cache.scriptHandlerFactory
}
