package testutils

import (
	"net/http"

	"github.com/evg4b/uncors/internal/contracts"
)

type handlerFunc = func(writer contracts.ResponseWriter, request *contracts.Request)

type CountableHandler struct {
	handler handlerFunc
	count   int
}

func NewCounter(handler handlerFunc) *CountableHandler {
	return &CountableHandler{handler, 0}
}

func (t *CountableHandler) ServeHTTP(writer contracts.ResponseWriter, request *contracts.Request) {
	t.count++
	t.handler(writer, request)
}

func (t *CountableHandler) Count() int {
	return t.count
}

func CopyHeaders(from http.Header, to http.Header) {
	for key, values := range from {
		for _, value := range values {
			to.Add(key, value)
		}
	}
}
