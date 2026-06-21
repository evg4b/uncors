package uncors

import (
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler"
	"github.com/evg4b/uncors/internal/handler/har"
	"github.com/evg4b/uncors/internal/handler/mock"
	"github.com/evg4b/uncors/internal/handler/proxy"
	"github.com/evg4b/uncors/internal/handler/rewrite"
	"github.com/evg4b/uncors/internal/handler/script"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/tui/styles"
	"github.com/evg4b/uncors/internal/urlreplacer"
)

func (app *Uncors) buildProxyHandler(proxyURL string, mappings config.Mappings) contracts.Handler {
	prefix := styles.ProxyStyle.Render("PROXY")

	return infra.WithPrefix(prefix, proxy.NewProxyHandler(
		proxy.WithURLReplacerFactory(urlreplacer.NewURLReplacerFactory(mappings)),
		proxy.WithHTTPClient(infra.MakeHTTPClient(proxyURL)),
		proxy.WithOutput(app.output.NewPrefixOutput(prefix)),
	))
}

func (app *Uncors) buildCacheMiddlewareFactory(cfg *config.CacheConfig) handler.CacheMiddlewareFactory {
	return func(globs config.CacheGlobs) contracts.Middleware {
		return infra.NewPrefixedMiddleware(
			app.container.CacheMiddleware(cfg, globs),
			styles.CacheStyle.Render("CACHE"),
		)
	}
}

func (app *Uncors) buildOptionsMiddlewareFactory() handler.OptionsMiddlewareFactory {
	return func(cfg config.OptionsHandling) contracts.Middleware {
		return infra.NewPrefixedMiddleware(
			app.container.OptionsMiddleware(cfg),
			styles.OptionsStyle.Render("OPTIONS"),
		)
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
		return infra.NewPrefixedMiddleware(
			app.container.StaticMiddleware(path, dir),
			styles.StaticStyle.Render("STATIC"),
		)
	}
}

func (app *Uncors) buildMockHandlerFactory() handler.MockHandlerFactory {
	return func(response config.Response) contracts.Handler {
		prefix := styles.MockStyle.Render("MOCK")

		return infra.WithPrefix(prefix, mock.NewMockHandler(
			mock.WithResponse(response),
			mock.WithFileSystem(app.fs),
			mock.WithAfter(time.After),
		))
	}
}

func (app *Uncors) buildScriptHandlerFactory() handler.ScriptHandlerFactory {
	return func(scriptConfig config.Script) contracts.Handler {
		prefix := styles.RewriteStyle.Render("SCRIPT")

		return infra.WithPrefix(prefix, script.NewHandler(
			script.WithOutput(app.output.NewPrefixOutput(prefix)),
			script.WithScript(scriptConfig),
			script.WithFileSystem(app.fs),
		))
	}
}

func (app *Uncors) buildRewriteMiddlewareFactory() handler.RewriteMiddlewareFactory {
	return func(rewriting config.RewritingOption) contracts.Middleware {
		return infra.NewPrefixedMiddleware(
			rewrite.NewMiddleware(rewrite.WithRewritingOptions(rewriting)),
			styles.RewriteStyle.Render("REWRITE"),
		)
	}
}
