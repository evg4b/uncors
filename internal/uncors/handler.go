package uncors

import (
	"context"
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
	)
}

func (app *Uncors) buildProxyHandler(uncorsConfig *config.UncorsConfig, mappings config.Mappings) contracts.Handler {
	prefix := styles.ProxyStyle.Render("PROXY")

	return withPrefix(prefix, handler.LazyHandler(func() contracts.Handler {
		return proxy.NewProxyHandler(
			proxy.WithURLReplacerFactory(urlreplacer.NewURLReplacerFactory(mappings)),
			proxy.WithHTTPClient(infra.MakeHTTPClient(uncorsConfig.Proxy)),
			proxy.WithOutput(app.output.NewPrefixOutput(prefix)),
		)
	}))
}

func (app *Uncors) buildCacheMiddlewareFactory(cfg config.CacheConfig) handler.CacheMiddlewareFactory {
	return func(globs config.CacheGlobs) contracts.Middleware {
		prefix := styles.CacheStyle.Render("CACHE")

		return handler.MiddlewareFunc(func(next contracts.Handler) contracts.Handler {
			return withPrefix(prefix, handler.LazyMiddleware(func() contracts.Middleware {
				return cache.NewMiddleware(
					cache.WithOutput(app.output.NewPrefixOutput(prefix)),
					cache.WithMethods(cfg.Methods),
					cache.WithCacheStorage(app.getCacheStorage(cfg)),
					cache.WithGlobs(globs),
				)
			}).Wrap(next))
		})
	}
}

func (app *Uncors) buildOptionsMiddlewareFactory() handler.OptionsMiddlewareFactory {
	return func(cfg config.OptionsHandling) contracts.Middleware {
		prefix := styles.OptionsStyle.Render("OPTIONS")

		return handler.MiddlewareFunc(func(next contracts.Handler) contracts.Handler {
			return withPrefix(prefix, handler.LazyMiddleware(func() contracts.Middleware {
				return options.NewMiddleware(
					options.WithOutput(app.output.NewPrefixOutput(prefix)),
					options.WithHeaders(cfg.Headers),
					options.WithCode(cfg.Code),
				)
			}).Wrap(next))
		})
	}
}

func (app *Uncors) buildStaticMiddlewareFactory() handler.StaticMiddlewareFactory {
	return func(path string, dir config.StaticDirectory) contracts.Middleware {
		prefix := styles.StaticStyle.Render("STATIC")

		return handler.MiddlewareFunc(func(next contracts.Handler) contracts.Handler {
			return withPrefix(prefix, handler.LazyMiddleware(func() contracts.Middleware {
				return static.NewStaticMiddleware(
					static.WithFileSystem(afero.NewBasePathFs(app.fs, dir.Dir)),
					static.WithIndex(dir.Index),
					static.WithOutput(mocks.NoopOutput()),
					static.WithPrefix(path),
				)
			}).Wrap(next))
		})
	}
}

func (app *Uncors) buildMockHandlerFactory() handler.MockHandlerFactory {
	return func(response config.Response) contracts.Handler {
		prefix := styles.MockStyle.Render("MOCK")

		return withPrefix(prefix, handler.LazyHandler(func() contracts.Handler {
			return mock.NewMockHandler(
				mock.WithOutput(app.output.NewPrefixOutput(prefix)),
				mock.WithResponse(response),
				mock.WithFileSystem(app.fs),
				mock.WithAfter(time.After),
			)
		}))
	}
}

func (app *Uncors) buildScriptHandlerFactory() handler.ScriptHandlerFactory {
	return func(scriptConfig config.Script) contracts.Handler {
		prefix := styles.RewriteStyle.Render("SCRIPT")

		return withPrefix(prefix, handler.LazyHandler(func() contracts.Handler {
			return script.NewHandler(
				script.WithOutput(app.output.NewPrefixOutput(prefix)),
				script.WithScript(scriptConfig),
				script.WithFileSystem(app.fs),
			)
		}))
	}
}

func (app *Uncors) buildRewriteMiddlewareFactory() handler.RewriteMiddlewareFactory {
	return func(rewriting config.RewritingOption) contracts.Middleware {
		prefix := styles.RewriteStyle.Render("REWRITE")

		return handler.MiddlewareFunc(func(next contracts.Handler) contracts.Handler {
			return withPrefix(prefix, handler.LazyMiddleware(func() contracts.Middleware {
				return rewrite.NewMiddleware(rewrite.WithRewritingOptions(rewriting))
			}).Wrap(next))
		})
	}
}

func withPrefix(prefix string, next contracts.Handler) contracts.Handler {
	return contracts.HandlerFunc(func(resp contracts.ResponseWriter, req *contracts.Request) {
		if updater, ok := req.Context().Value(contracts.PrefixUpdaterKey).(func(string)); ok {
			updater(prefix)
		}

		ctx := context.WithValue(req.Context(), contracts.PrefixKey, prefix)
		next.ServeHTTP(resp, req.WithContext(ctx))
	})
}
