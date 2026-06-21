package uncors

import (
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler"
	"github.com/evg4b/uncors/internal/handler/proxy"
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
		return app.container.CacheMiddleware(cfg, globs)
	}
}
