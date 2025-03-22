package handler

import (
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
)

func (h *RequestHandler) wrapCacheMiddleware(cache config.CacheGlobs, next contracts.Handler) contracts.Handler {
	if len(cache) > 0 {
		cacheMiddleware := h.cacheMiddlewareFactory(cache)

		return cacheMiddleware.Wrap(next)
	}

	return next
}
