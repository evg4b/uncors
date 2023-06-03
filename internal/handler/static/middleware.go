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
	next   http.Handler
	index  string
	logger contracts.Logger
	prefix string
}

func NewStaticMiddleware(options ...MiddlewareOption) *Middleware {
	middleware := &Middleware{}

	for _, option := range options {
		option(middleware)
	}

	return middleware
}

func (m *Middleware) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	filePath := m.extractFilePath(request)

	file, stat, err := m.openFile(filePath)
	defer helpers.CloseSafe(file)

	if err != nil {
		if errors.Is(err, errNorHandled) {
			m.next.ServeHTTP(writer, request)
		} else {
			infra.HTTPError(writer, err)
		}

		return
	}

	http.ServeContent(writer, request, stat.Name(), stat.ModTime(), file)
}

func (m *Middleware) extractFilePath(request *http.Request) string {
	filePath := strings.TrimPrefix(request.URL.Path, m.prefix)
	if !strings.HasPrefix(filePath, "/") {
		filePath = "/" + filePath
	}

	return path.Clean(filePath)
}
