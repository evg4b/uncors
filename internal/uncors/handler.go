package uncors

import (
	"time"

	"github.com/evg4b/uncors/pkg/fakedata"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler"
	cache2 "github.com/evg4b/uncors/internal/handler/cache"
	"github.com/evg4b/uncors/internal/handler/mock"
	"github.com/evg4b/uncors/internal/handler/proxy"
	"github.com/evg4b/uncors/internal/handler/static"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/patrickmn/go-cache"
	"github.com/spf13/afero"
)

type appCache struct {
	staticHandlerFactory handler.RequestHandlerOption
	mockHandlerFactory   handler.RequestHandlerOption
}

func (app *App) buildHandler(uncorsConfig *config.UncorsConfig) *handler.RequestHandler {
	globalHandler := handler.NewUncorsRequestHandler(
		handler.WithMappings(uncorsConfig.Mappings),
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
		handler.WithProxyHandlerFactory(func() contracts.Handler {
			factory := urlreplacer.NewURLReplacerFactory(uncorsConfig.Mappings)
			httpClient := infra.MakeHTTPClient(uncorsConfig.Proxy)

			return proxy.NewProxyHandler(
				proxy.WithURLReplacerFactory(factory),
				proxy.WithHTTPClient(httpClient),
				proxy.WithLogger(NewProxyLogger(app.logger)),
			)
		}),
		app.getWithStaticHandlerFactory(),
		app.getMockHandlerFactory(),
	)

	return globalHandler
}

func (app *App) getMockHandlerFactory() handler.RequestHandlerOption {
	if app.cache.mockHandlerFactory == nil {
		factoryFunc := func(response config.Response) contracts.Handler {
			return mock.NewMockHandler(
				mock.WithLogger(NewMockLogger(app.logger)),
				mock.WithResponse(response),
				mock.WithFileSystem(app.fs),
				mock.WithAfter(time.After),
				mock.WithGenerator(fakedata.NewGoFakeItGenerator()),
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
