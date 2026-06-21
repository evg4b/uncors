package uncors

import (
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler"
)

func (app *Uncors) buildCacheMiddlewareFactory(cfg *config.CacheConfig) handler.CacheMiddlewareFactory {
	return func(globs config.CacheGlobs) contracts.Middleware {
		return app.container.CacheMiddleware(cfg, globs)
	}
}
