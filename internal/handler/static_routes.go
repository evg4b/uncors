package handler

import (
	"net/http"
	"strings"

	"github.com/evg4b/uncors/internal/ui"

	"github.com/evg4b/uncors/internal/handler/static"

	"github.com/spf13/afero"
)

func (m *UncorsRequestHandler) makeStaticRoutes(next http.Handler) {
	for _, urlMapping := range m.mappings {
		for _, staticDir := range urlMapping.Statics {
			clearPath := strings.TrimSuffix(staticDir.Path, "/")
			path := clearPath + "/"

			redirect := m.router.NewRoute()
			redirect.Path(clearPath).
				Handler(http.RedirectHandler(path, http.StatusTemporaryRedirect))

			route := m.router.NewRoute()
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
}
