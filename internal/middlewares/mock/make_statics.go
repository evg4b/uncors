package mock

import (
	"net/http"
	"strings"

	"github.com/spf13/afero"
)

func (m *Middleware) makeStaticRoutes() {
	for _, mapping := range m.mappings {
		for _, staticDirMapping := range mapping.Statics {
			path := strings.TrimRight(staticDirMapping.Path, "/")
			path2 := path + "/"

			redirect := m.router.NewRoute()
			redirect.Path(path).
				Handler(http.RedirectHandler(path2, http.StatusTemporaryRedirect))

			route := m.router.NewRoute()
			fs := afero.NewBasePathFs(m.fs, staticDirMapping.Dir)
			fileServer := http.FileServer(afero.NewHttpFs(fs))
			handler := http.StripPrefix(path2, fileServer)

			route.PathPrefix(path2).
				Handler(handler)
		}
	}
}
