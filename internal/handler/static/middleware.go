package static

import (
	"errors"
	"net/http"
	"path"
	"strings"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/spf13/afero"
)

type Middleware struct {
	fs     afero.Fs
	index  string
	logger contracts.Logger
	prefix string
}

func NewStaticMiddleware(options ...MiddlewareOption) *Middleware {
	return helpers.ApplyOptions(&Middleware{}, options)
}

func (h *Middleware) Wrap(next contracts.Handler) contracts.Handler {
	return contracts.HandlerFunc(func(writer contracts.ResponseWriter, request *contracts.Request) {
		response := contracts.WrapResponseWriter(writer)

		filePath := h.extractFilePath(request)
		file, stat, err := h.openFile(filePath)
		defer helpers.CloseSafe(file)

		if err != nil {
			if errors.Is(err, errNorHandled) {
				next.ServeHTTP(response, request)
			} else {
				infra.HTTPError(response, err)
			}

			return
		}

		http.ServeContent(response, request, stat.Name(), stat.ModTime(), file)
	})
}

func (h *Middleware) extractFilePath(request *http.Request) string {
	filePath := strings.TrimPrefix(request.URL.Path, h.prefix)
	if !strings.HasPrefix(filePath, "/") {
		filePath = "/" + filePath
	}

	return path.Clean(filePath)
}
