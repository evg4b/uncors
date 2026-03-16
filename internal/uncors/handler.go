package uncors

import (
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler"
	"github.com/evg4b/uncors/internal/handler/cache"
	"github.com/evg4b/uncors/internal/handler/har"
	"github.com/evg4b/uncors/internal/handler/mock"
	"github.com/evg4b/uncors/internal/handler/options"
	"github.com/evg4b/uncors/internal/handler/proxy"
	"github.com/evg4b/uncors/internal/handler/rewrite"
	"github.com/evg4b/uncors/internal/handler/script"
	"github.com/evg4b/uncors/internal/handler/static"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/tui/styles"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/evg4b/uncors/testing/mocks"
	"github.com/spf13/afero"
)

func (app *Uncors) buildHandlerForMappings(
	config *config.UncorsConfig, mappings config.Mappings,
) *handler.RequestHandler {
	return handler.NewUncorsRequestHandler(
		handler.WithMappings(mappings),
		handler.WithProxyHandler(app.buildProxyHandler(config, mappings)),
		handler.WithCacheMiddlewareFactory(app.buildCacheMiddlewareFactory(config.CacheConfig)),
		handler.WithOptionsHandlerFactory(app.buildOptionsMiddlewareFactory()),
		handler.WithStaticHandlerFactory(app.buildStaticMiddlewareFactory()),
		handler.WithMockHandlerFactory(app.buildMockHandlerFactory()),
		handler.WithScriptHandlerFactory(app.buildScriptHandlerFactory()),
		handler.WithRewriteHandlerFactory(app.buildRewriteMiddlewareFactory()),
		handler.WithOutput(app.output),
		handler.WithHARMiddlewareFactory(app.buildHARMiddlewareFactory()),
	)
}

func (app *Uncors) buildProxyHandler(uncorsConfig *config.UncorsConfig, mappings config.Mappings) contracts.Handler {
	return handler.LazyHandler(func() contracts.Handler {
		return proxy.NewProxyHandler(
			proxy.WithURLReplacerFactory(urlreplacer.NewURLReplacerFactory(mappings)),
			proxy.WithHTTPClient(infra.MakeHTTPClient(uncorsConfig.Proxy)),
			proxy.WithOutput(app.output.NewPrefixOutput(styles.ProxyStyle.Render("PROXY"))),
		)
	})
}

func (app *Uncors) buildCacheMiddlewareFactory(cfg config.CacheConfig) handler.CacheMiddlewareFactory {
	return func(globs config.CacheGlobs) contracts.Middleware {
		return handler.LazyMiddleware(func() contracts.Middleware {
			return cache.NewMiddleware(
				cache.WithOutput(app.output.NewPrefixOutput(styles.CacheStyle.Render("CACHE"))),
				cache.WithMethods(cfg.Methods),
				cache.WithCacheStorage(app.getCacheStorage(cfg)),
				cache.WithGlobs(globs),
			)
		})
	}
}

func (app *Uncors) buildOptionsMiddlewareFactory() handler.OptionsMiddlewareFactory {
	return func(cfg config.OptionsHandling) contracts.Middleware {
		return handler.LazyMiddleware(func() contracts.Middleware {
			return options.NewMiddleware(
				options.WithOutput(app.output.NewPrefixOutput(styles.OptionsStyle.Render("OPTIONS"))),
				options.WithHeaders(cfg.Headers),
				options.WithCode(cfg.Code),
			)
		})
	}
}

func (app *Uncors) buildHARMiddlewareFactory() handler.HARMiddlewareFactory {
	return func(harConfig config.HARConfig) contracts.Middleware {
		return handler.LazyMiddleware(func() contracts.Middleware {
			w := har.NewWriter(harConfig.File)
			app.registerCloser(w)

			return har.NewMiddleware(
				har.WithWriter(w),
				har.WithCaptureSecureHeaders(harConfig.CaptureSecureHeaders),
			)
		})
	}
}

func (app *Uncors) buildStaticMiddlewareFactory() handler.StaticMiddlewareFactory {
	return func(path string, dir config.StaticDirectory) contracts.Middleware {
		return handler.LazyMiddleware(func() contracts.Middleware {
			return static.NewStaticMiddleware(
				static.WithFileSystem(afero.NewBasePathFs(app.fs, dir.Dir)),
				static.WithIndex(dir.Index),
				static.WithOutput(mocks.NoopOutput()),
				static.WithPrefix(path),
			)
		})
	}
}

func (app *Uncors) buildMockHandlerFactory() handler.MockHandlerFactory {
	return func(response config.Response) contracts.Handler {
		return handler.LazyHandler(func() contracts.Handler {
			return mock.NewMockHandler(
				mock.WithOutput(app.output.NewPrefixOutput(styles.MockStyle.Render("MOCK"))),
				mock.WithResponse(response),
				mock.WithFileSystem(app.fs),
				mock.WithAfter(time.After),
			)
		})
	}
}

func (app *Uncors) buildScriptHandlerFactory() handler.ScriptHandlerFactory {
	return func(s config.Script) contracts.Handler {
		return handler.LazyHandler(func() contracts.Handler {
			return script.NewHandler(
				script.WithOutput(app.output.NewPrefixOutput(styles.RewriteStyle.Render("SCRIPT"))),
				script.WithScript(s),
				script.WithFileSystem(app.fs),
			)
		})
	}
}

func (app *Uncors) buildRewriteMiddlewareFactory() handler.RewriteMiddlewareFactory {
	return func(rewriting config.RewritingOption) contracts.Middleware {
		return handler.LazyMiddleware(func() contracts.Middleware {
			return rewrite.NewMiddleware(rewrite.WithRewritingOptions(rewriting))
		})
	}
}
