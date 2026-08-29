package infra

import (
	"net/http"
)

// HandlerFunc adapts an error returning function to http.Handler: a returned
// error is rendered as an HTTP error response. Handlers opt into error returns
// where they benefit from them, while the pipeline itself stays plain net/http.
type HandlerFunc func(http.ResponseWriter, *http.Request) error

func (f HandlerFunc) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	err := f(writer, request)
	if err != nil {
		HTTPError(writer, err)
	}
}
