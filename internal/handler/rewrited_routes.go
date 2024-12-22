package handler

import (
	"net/http"
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

		redirect := router.NewRoute()
		redirect.Path(clearPath).
			Handler(http.RedirectHandler(path, http.StatusTemporaryRedirect))

		route := router.NewRoute()

		route.PathPrefix(path).
			Handler(
				contracts.CastToHTTPHandler(
					h.rewriteHandlerFactory(rewrite),
				),
			)
	}
}
