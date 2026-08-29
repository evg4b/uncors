package testutils

import (
	"net/http"

	"github.com/gorilla/mux"
)

type CountableHandler struct {
	handler http.Handler
	count   int
}

func NewCounter(handler http.Handler) *CountableHandler {
	return &CountableHandler{handler, 0}
}

func (t *CountableHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	t.count++

	t.handler.ServeHTTP(writer, request)
}

func (t *CountableHandler) Count() int {
	return t.count
}

func (t *CountableHandler) Reset() {
	t.count = 0
}

func CopyHeaders(from http.Header, to http.Header) {
	for key, values := range from {
		for _, value := range values {
			to.Add(key, value)
		}
	}
}

// SetMuxVars sets gorilla/mux route variables for testing.
func SetMuxVars(req *http.Request, vars map[string]string) *http.Request {
	return mux.SetURLVars(req, vars)
}
