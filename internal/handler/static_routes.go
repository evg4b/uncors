package handler

import (
	"net/http"
	"strings"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/handler/static"
	"github.com/evg4b/uncors/internal/ui"
	"github.com/gorilla/mux"
	"github.com/spf13/afero"
)

func (m *RequestHandler) makeStaticRoutes(router *mux.Router, statics config.StaticDirs, next http.Handler) {
	for _, staticDir := range statics {
		clearPath := strings.TrimSuffix(staticDir.Path, "/")
		path := clearPath + "/"

		redirect := router.NewRoute()
		redirect.Path(clearPath).
			Handler(http.RedirectHandler(path, http.StatusTemporaryRedirect))

		route := router.NewRoute()
		handler := static.NewStaticMiddleware(
			static.WithFileSystem(afero.NewBasePathFs(m.fs, staticDir.Dir)),
			static.WithIndex(staticDir.Index),
			static.WithNext(next),
			static.WithLogger(ui.StaticLogger),
			static.WithPrefix(path),
		)

		route.PathPrefix(path).Handler(handler)
	}
}
