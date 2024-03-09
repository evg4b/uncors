package static

import (
	"errors"
	"net/http"
	"path"
	"strings"

	"github.com/evg4b/uncors/internal/tui/request_tracker"
	"github.com/evg4b/uncors/internal/tui/styles"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/spf13/afero"
)

type Middleware struct {
	fs      afero.Fs
	index   string
	logger  contracts.Logger
	prefix  string
	tracker request_tracker.RequestTracker
}

func NewStaticMiddleware(options ...MiddlewareOption) *Middleware {
	return helpers.ApplyOptions(&Middleware{}, options)
}

func (h *Middleware) Wrap(next contracts.Handler) contracts.Handler {
	return contracts.HandlerFunc(func(response contracts.ResponseWriter, request *contracts.Request) {
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

		handler := contracts.HandlerFunc(func(response contracts.ResponseWriter, request *contracts.Request) {
			http.ServeContent(response, request, stat.Name(), stat.ModTime(), file)
		})

		h.tracker.Wrap(handler, styles.StaticStyle.Render("STATIC")).
			ServeHTTP(response, request)
	})
}

func (h *Middleware) extractFilePath(request *http.Request) string {
	filePath := strings.TrimPrefix(request.URL.Path, h.prefix)
	if !strings.HasPrefix(filePath, "/") {
		filePath = "/" + filePath
	}

	return path.Clean(filePath)
}
