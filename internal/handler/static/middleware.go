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
	next   contracts.Handler
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

func (m *Middleware) ServeHTTP(writer *contracts.ResponseWriter, request *contracts.Request) {
	response := contracts.WrapResponseWriter(writer)

	filePath := m.extractFilePath(request)
	file, stat, err := m.openFile(filePath)
	defer helpers.CloseSafe(file)

	if err != nil {
		if errors.Is(err, errNorHandled) {
			m.next.ServeHTTP(response, request)
		} else {
			infra.HTTPError(response, err)
		}

		return
	}

	http.ServeContent(response, request, stat.Name(), stat.ModTime(), file)
	m.logger.PrintResponse(&http.Response{
		StatusCode: response.StatusCode,
		Request:    request,
	})
}

func (m *Middleware) extractFilePath(request *http.Request) string {
	filePath := strings.TrimPrefix(request.URL.Path, m.prefix)
	if !strings.HasPrefix(filePath, "/") {
		filePath = "/" + filePath
	}

	return path.Clean(filePath)
}
