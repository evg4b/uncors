package static

import (
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/spf13/afero"
)

type Middleware struct {
	fs    afero.Fs
	next  http.Handler
	index string
}

var errNorHandled = errors.New("request is not handled")

func NewStaticMiddleware(options ...MiddlewareOption) *Middleware {
	middleware := &Middleware{
		index: "index.html",
	}

	for _, option := range options {
		option(middleware)
	}

	return middleware
}

func toHTTPError(err error) (string, int) {
	if errors.Is(err, fs.ErrNotExist) {
		return http.StatusText(http.StatusNotFound), http.StatusNotFound
	}
	if errors.Is(err, fs.ErrPermission) {
		return http.StatusText(http.StatusForbidden), http.StatusForbidden
	}

	return http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError
}

func (m *Middleware) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	upath := request.URL.Path
	if !strings.HasPrefix(upath, "/") {
		upath = "/" + upath
		request.URL.Path = upath
	}

	m.serveFile(writer, request, path.Clean(upath))
}

func (m *Middleware) serveFile(writer http.ResponseWriter, request *http.Request, name string) {
	file, stat, err := m.openFile(name)
	if err != nil {
		if errors.Is(err, errNorHandled) {
			m.next.ServeHTTP(writer, request)
		} else {
			msg, code := toHTTPError(err)
			http.Error(writer, msg, code)
		}
	}
	defer file.Close()

	http.ServeContent(writer, request, stat.Name(), stat.ModTime(), file)
}

func (m *Middleware) openFile(name string) (afero.File, os.FileInfo, error) {
	file, err := m.fs.Open(name)
	if err != nil && errors.Is(err, fs.ErrNotExist) {
		if len(m.index) == 0 {
			return nil, nil, errNorHandled
		}

		indexFile, err := m.fs.Open(m.index)
		if err != nil {
			return nil, nil, err
		}

		file = indexFile
	}

	stat, err := file.Stat()
	if err != nil {
		return nil, nil, err
	}

	if stat.IsDir() {
		if len(m.index) == 0 {
			return nil, nil, errNorHandled
		}

		if indexFile, err := m.fs.Open(m.index); err == nil {
			indexFileStat, err := indexFile.Stat()
			if err != nil {
				return nil, nil, err
			}

			stat = indexFileStat
			file = indexFile
		}
	}

	return file, stat, nil
}
