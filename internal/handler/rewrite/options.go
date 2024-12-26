package rewrite

import "github.com/evg4b/uncors/internal/config"

type MiddlewareOption = func(*Middleware)

func WithRewritingOptions(rewrite config.RewritingOption) MiddlewareOption {
	return func(h *Middleware) {
		h.rewrite = rewrite
	}
}
