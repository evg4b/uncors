package handler

import (
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
)

func (h *RequestHandler) wrapHARMiddleware(harConfig config.HARConfig, next contracts.Handler) contracts.Handler {
	if !harConfig.Enabled() {
		return next
	}

	middleware := h.harMiddlewareFactory(harConfig)

	return middleware.Wrap(next)
}
