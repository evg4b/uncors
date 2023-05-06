package static

import (
	"net/http"

	"github.com/spf13/afero"
)

type Middleware struct {
	fs      afero.Fs
	prefix  string
	handler http.Handler
	next    http.Handler
}

func NewStaticMiddleware(options ...MiddlewareOption) *Middleware {
	middleware := &Middleware{}

	for _, option := range options {
		option(middleware)
	}

	fileServer := http.FileServer(afero.NewHttpFs(middleware.fs))
	middleware.handler = http.StripPrefix(middleware.prefix, fileServer)

	return middleware
}

func (m *Middleware) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	m.handler.ServeHTTP(writer, request)
}
