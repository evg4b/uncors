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

type Handler struct {
	fs     afero.Fs
	next   contracts.Handler
	index  string
	logger contracts.Logger
	prefix string
}

func NewStaticHandler(options ...HandlerOption) *Handler {
	handler := &Handler{}

	for _, option := range options {
		option(handler)
	}

	return handler
}

func (h *Handler) ServeHTTP(writer contracts.ResponseWriter, request *contracts.Request) {
	response := contracts.WrapResponseWriter(writer)

	filePath := h.extractFilePath(request)
	file, stat, err := h.openFile(filePath)
	defer helpers.CloseSafe(file)

	if err != nil {
		if errors.Is(err, errNorHandled) {
			h.next.ServeHTTP(response, request)
		} else {
			infra.HTTPError(response, err)
		}

		return
	}

	http.ServeContent(response, request, stat.Name(), stat.ModTime(), file)
	h.logger.PrintResponse(request, response.StatusCode())
}

func (h *Handler) extractFilePath(request *http.Request) string {
	filePath := strings.TrimPrefix(request.URL.Path, h.prefix)
	if !strings.HasPrefix(filePath, "/") {
		filePath = "/" + filePath
	}

	return path.Clean(filePath)
}
