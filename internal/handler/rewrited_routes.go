package handler

import (
	"strings"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/gorilla/mux"
)

func (h *RequestHandler) makeRewritedRoutes(
	router *mux.Router,
	rewrites config.RewriteOptions,
) {
	for _, rewrite := range rewrites {
		clearPath := strings.TrimSuffix(rewrite.From, "/")
		path := clearPath + "/"

		handler := contracts.CastToHTTPHandler(
			h.rewriteHandlerFactory(rewrite),
		)

		redirect := router.NewRoute()
		redirect.Path(clearPath).Handler(handler)
		route := router.NewRoute()
		route.PathPrefix(path).Handler(handler)
	}
}
