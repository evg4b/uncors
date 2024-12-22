package rewrite

import (
	"github.com/evg4b/uncors/internal/config"
)

type RewriteOption = func(*Handler)

func WithRewritingOptions(rewrite config.RewritingOption) RewriteOption {
	return func(h *Handler) {
		h.rewrite = rewrite
	}
}
