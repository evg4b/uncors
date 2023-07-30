package handler

import (
	"net/http"
	"strings"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/gorilla/mux"
)

func (h *RequestHandler) makeStaticRoutes(
	router *mux.Router,
	statics config.StaticDirectories,
	next contracts.Handler,
) {
	for _, staticDir := range statics {
		clearPath := strings.TrimSuffix(staticDir.Path, "/")
		path := clearPath + "/"

		redirect := router.NewRoute()
		redirect.Path(clearPath).
			Handler(http.RedirectHandler(path, http.StatusTemporaryRedirect))

		route := router.NewRoute()

		middleware := h.staticMiddlewareFactory(path, staticDir)
		httpHandler := contracts.CastToHTTPHandler(middleware.Wrap(next))

		route.PathPrefix(path).Handler(httpHandler)
	}
}
