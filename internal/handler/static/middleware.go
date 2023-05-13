package static

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/evg4b/uncors/internal/contracts"

	"github.com/spf13/afero"
)

type Middleware struct {
	fs     afero.Fs
	next   http.Handler
	index  string
	logger contracts.Logger
	prefix string
}

var errNorHandled = errors.New("request is not handled")

func NewStaticMiddleware(options ...MiddlewareOption) *Middleware {
	middleware := &Middleware{}

	for _, option := range options {
		option(middleware)
	}

	return middleware
}

func toHTTPError(err error) (string, int) {
	if errors.Is(err, fs.ErrPermission) {
		return http.StatusText(http.StatusForbidden), http.StatusForbidden
	}

	return http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError
}

func (m *Middleware) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	filePath := m.extractFilePath(request)

	file, stat, err := m.openFile(filePath)
	defer func() {
		if file != nil {
			file.Close()
		}
	}()

	if err != nil {
		if errors.Is(err, errNorHandled) {
			m.next.ServeHTTP(writer, request)
		} else {
			msg, code := toHTTPError(err)
			http.Error(writer, msg, code)
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

func (m *Middleware) openFile(filePath string) (afero.File, os.FileInfo, error) {
	file, err := m.fs.Open(filePath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, nil, fmt.Errorf("filed to open file: %w", err)
		}

		indexFile, err := m.openIndexFile()
		if err != nil {
			return nil, nil, err
		}

		file = indexFile
	}

	stat, err := file.Stat()
	if err != nil {
		return file, nil, fmt.Errorf("filed to get information about file: %w", err)
	}

	if stat.IsDir() {
		indexFile, err := m.openIndexFile()
		if err != nil {
			return file, stat, err
		}

		indexFileStat, err := indexFile.Stat()
		if err != nil {
			return file, stat, fmt.Errorf("filed to get information about index file: %w", err)
		}

		file = indexFile
		stat = indexFileStat
	}

	return file, stat, nil
}

func (m *Middleware) openIndexFile() (afero.File, error) {
	if len(m.index) == 0 {
		return nil, errNorHandled
	}

	file, err := m.fs.Open(m.index)
	if err != nil {
		return nil, fmt.Errorf("filed to opend index file: %w", err)
	}

	return file, nil
}
