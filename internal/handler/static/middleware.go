package static

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/spf13/afero"
)

var errNotHandled = errors.New("request is not handled")

type Middleware struct {
	fs     afero.Fs
	index  string
	output contracts.Output
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
			if errors.Is(err, errNotHandled) {
				next.ServeHTTP(response, request)
			} else {
				log.Printf("ERROR: Static handler error: %v, url: %s", err, request.URL)
				infra.HTTPError(response, err)
			}

			return
		}

		http.ServeContent(response, request, stat.Name(), stat.ModTime(), file)
		h.output.Request(helpers.ToRequestData(request, response))
	})
}

func (h *Middleware) extractFilePath(request *http.Request) string {
	filePath := strings.TrimPrefix(request.URL.Path, h.prefix)
	if !strings.HasPrefix(filePath, "/") {
		filePath = "/" + filePath
	}

	return path.Clean(filePath)
}

func (h *Middleware) openFile(filePath string) (afero.File, os.FileInfo, error) {
	file, err := h.fs.Open(filePath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, nil, fmt.Errorf("failed to open file: %w", err)
		}

		indexFile, err := h.openIndexFile()
		if err != nil {
			return nil, nil, err
		}

		file = indexFile
	}

	stat, err := file.Stat()
	if err != nil {
		return file, nil, fmt.Errorf("failed to get information about file: %w", err)
	}

	if stat.IsDir() {
		indexFile, err := h.openIndexFile()
		if err != nil {
			return file, stat, err
		}

		indexFileStat, err := indexFile.Stat()
		if err != nil {
			return file, stat, fmt.Errorf("failed to get information about index file: %w", err)
		}

		file = indexFile
		stat = indexFileStat
	}

	return file, stat, nil
}

func (h *Middleware) openIndexFile() (afero.File, error) {
	if len(h.index) == 0 {
		return nil, errNotHandled
	}

	file, err := h.fs.Open(h.index)
	if err != nil {
		return nil, fmt.Errorf("failed to open index file: %w", err)
	}

	return file, nil
}

type MiddlewareOption = func(*Middleware)

func WithFileSystem(fs afero.Fs) MiddlewareOption {
	return func(h *Middleware) {
		h.fs = fs
	}
}

func WithIndex(index string) MiddlewareOption {
	return func(h *Middleware) {
		h.index = index
	}
}

func WithOutput(output contracts.Output) MiddlewareOption {
	return func(h *Middleware) {
		h.output = output
	}
}

func WithPrefix(prefix string) MiddlewareOption {
	return func(h *Middleware) {
		h.prefix = prefix
	}
}
