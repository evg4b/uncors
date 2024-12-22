package rewrite

import (
	"fmt"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
)

type Handler struct {
	rewrite config.RewritingOption
}

func NewHandler(options ...RewriteOption) *Handler {
	return helpers.ApplyOptions(&Handler{}, options)
}

func (h *Handler) Wrap(next contracts.Handler) contracts.Handler {
	return contracts.HandlerFunc(func(writer contracts.ResponseWriter, request *contracts.Request) {
		next.ServeHTTP(writer, request)
	})
}

func (h *Handler) ServeHTTP(writer contracts.ResponseWriter, request *contracts.Request) {
	fmt.Fprintf(writer, "Rewrite handler: %s -> %s\n", h.rewrite.From, h.rewrite.To)
}
