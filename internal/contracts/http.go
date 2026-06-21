package contracts

import (
	"net/http"
	"net/url"
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

type Next func(writer ResponseWriter, request *Request) error

type Middleware interface {
	ServeHTTP(writer ResponseWriter, request *Request, next Next) error
}
