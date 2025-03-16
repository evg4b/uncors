package handler

import (
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
)

func (h *RequestHandler) wrapOptionsMiddleware(
	options config.OptionsHandling,
	next contracts.Handler,
) contracts.Handler {
	if options.Disabled {
		return next
	}

	middleware := h.optionsMiddlewareFactory(options)

	return middleware.Wrap(next)
}
