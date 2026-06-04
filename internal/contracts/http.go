package contracts

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/evg4b/uncors/internal/infra"
)

type contextKey string

const (
	PrefixKey        contextKey = "uncors-prefix"
	PrefixUpdaterKey contextKey = "uncors-prefix-updater"
)

type RequestData struct {
	Method    string
	URL       *url.URL
	Header    http.Header
	Body      []byte
	Code      int
	Cancelled bool
}

type Request = http.Request

type Handler interface {
	ServeHTTP(writer ResponseWriter, request *Request) error
}

type Middleware interface {
	Wrap(next Handler) Handler
}

type HandlerFunc func(ResponseWriter, *Request) error

func (f HandlerFunc) ServeHTTP(w ResponseWriter, r *Request) error {
	return f(w, r)
}

var ErrResponseNotCasted = errors.New("received incorrect response writer type")

func CastToHTTPHandler(handler Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		writer, ok := response.(ResponseWriter)
		if !ok {
			panic(ErrResponseNotCasted)
		}

		err := handler.ServeHTTP(writer, request)
		if err != nil {
			infra.HTTPError(writer, err)
		}
	})
}

func CastToContractsHandler(handler http.Handler) Handler {
	return HandlerFunc(func(writer ResponseWriter, request *Request) error {
		handler.ServeHTTP(writer, request)

		return nil
	})
}
