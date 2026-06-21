package uncors

import (
	"context"
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
	"github.com/spf13/afero"
)

func (app *Uncors) buildProxyHandler(proxyURL string, mappings config.Mappings) contracts.Handler {
	prefix := styles.ProxyStyle.Render("PROXY")

	return withPrefix(prefix, proxy.NewProxyHandler(
		proxy.WithURLReplacerFactory(urlreplacer.NewURLReplacerFactory(mappings)),
		proxy.WithHTTPClient(infra.MakeHTTPClient(proxyURL)),
		proxy.WithOutput(app.output.NewPrefixOutput(prefix)),
	))
}

func (app *Uncors) buildCacheMiddlewareFactory(cfg config.CacheConfig) handler.CacheMiddlewareFactory {
	return func(globs config.CacheGlobs) contracts.Middleware {
		prefix := styles.CacheStyle.Render("CACHE")

		middleware := cache.NewMiddleware(
			cache.WithMethods(cfg.Methods),
			cache.WithCacheStorage(app.getCacheStorage(cfg)),
			cache.WithGlobs(globs),
		)

		return &prefixedMiddleware{
			middleware: middleware,
			prefix:     prefix,
		}
	}
}

func (app *Uncors) buildOptionsMiddlewareFactory() handler.OptionsMiddlewareFactory {
	return func(cfg config.OptionsHandling) contracts.Middleware {
		prefix := styles.OptionsStyle.Render("OPTIONS")

		middleware := options.NewMiddleware(
			options.WithHeaders(cfg.Headers),
			options.WithCode(cfg.Code),
		)

		return &prefixedMiddleware{
			middleware: middleware,
			prefix:     prefix,
		}
	}
}

func (app *Uncors) buildHARMiddlewareFactory() handler.HARMiddlewareFactory {
	return func(harConfig config.HARConfig) contracts.Middleware {
		w := har.NewWriter(harConfig.File)
		app.registerCloser(w)

		return har.NewMiddleware(
			har.WithWriter(w),
			har.WithCaptureSecureHeaders(harConfig.CaptureSecureHeaders),
		)
	}
}

func (app *Uncors) buildStaticMiddlewareFactory() handler.StaticMiddlewareFactory {
	return func(path string, dir config.StaticDirectory) contracts.Middleware {
		prefix := styles.StaticStyle.Render("STATIC")

		middleware := static.NewStaticMiddleware(
			static.WithFileSystem(afero.NewBasePathFs(app.fs, dir.Dir)),
			static.WithIndex(dir.Index),
			static.WithPrefix(path),
		)

		return &prefixedMiddleware{
			middleware: middleware,
			prefix:     prefix,
		}
	}
}

func (app *Uncors) buildMockHandlerFactory() handler.MockHandlerFactory {
	return func(response config.Response) contracts.Handler {
		prefix := styles.MockStyle.Render("MOCK")

		return withPrefix(prefix, mock.NewMockHandler(
			mock.WithResponse(response),
			mock.WithFileSystem(app.fs),
			mock.WithAfter(time.After),
		))
	}
}

func (app *Uncors) buildScriptHandlerFactory() handler.ScriptHandlerFactory {
	return func(scriptConfig config.Script) contracts.Handler {
		prefix := styles.RewriteStyle.Render("SCRIPT")

		return withPrefix(prefix, script.NewHandler(
			script.WithOutput(app.output.NewPrefixOutput(prefix)),
			script.WithScript(scriptConfig),
			script.WithFileSystem(app.fs),
		))
	}
}

func (app *Uncors) buildRewriteMiddlewareFactory() handler.RewriteMiddlewareFactory {
	return func(rewriting config.RewritingOption) contracts.Middleware {
		prefix := styles.RewriteStyle.Render("REWRITE")

		middleware := rewrite.NewMiddleware(rewrite.WithRewritingOptions(rewriting))

		return &prefixedMiddleware{
			middleware: middleware,
			prefix:     prefix,
		}
	}
}

func withPrefix(prefix string, next contracts.Handler) contracts.Handler {
	return contracts.HandlerFunc(func(resp contracts.ResponseWriter, req *contracts.Request) error {
		if updater, ok := req.Context().Value(contracts.PrefixUpdaterKey).(func(string)); ok {
			updater(prefix)
		}

		ctx := context.WithValue(req.Context(), contracts.PrefixKey, prefix)

		return next.ServeHTTP(resp, req.WithContext(ctx))
	})
}

type prefixedMiddleware struct {
	middleware contracts.Middleware
	prefix     string
}

func (p *prefixedMiddleware) ServeHTTP(w contracts.ResponseWriter, r *contracts.Request, next contracts.Next) error {
	return p.middleware.ServeHTTP(w, r, func(w contracts.ResponseWriter, r *contracts.Request) error {
		return withPrefix(p.prefix, contracts.HandlerFunc(func(w contracts.ResponseWriter, r *contracts.Request) error {
			return next(w, r)
		})).ServeHTTP(w, r)
	})
}
