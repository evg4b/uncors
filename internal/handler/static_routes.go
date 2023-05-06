package handler

import (
	"net/http"
	"strings"

	"github.com/evg4b/uncors/internal/handler/static"

	"github.com/spf13/afero"
)

func (m *UncorsRequestHandler) makeStaticRoutes(next http.Handler) {
	for _, mapping := range m.mappings {
		for _, staticDirMapping := range mapping.Statics {
			clearPath := strings.TrimSuffix(staticDirMapping.Path, "/")
			path := clearPath + "/"

			redirect := m.router.NewRoute()
			redirect.Path(clearPath).
				Handler(http.RedirectHandler(path, http.StatusTemporaryRedirect))

			route := m.router.NewRoute()
			handler := static.NewStaticMiddleware(
				static.WithFileSystem(afero.NewBasePathFs(m.fs, staticDirMapping.Dir)),
				static.WithIndex("index.html"),
				static.WithNext(next),
			)

			route.PathPrefix(path).
				Handler(http.StripPrefix(path, handler))
		}
	}
}
