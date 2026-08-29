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

// Middleware is the conventional Go middleware shape. Using the standard
// signature keeps the whole net/http ecosystem — http.TimeoutHandler,
// httputil.ReverseProxy, mux.Router.Use, tracing middleware — usable in the
// pipeline, and removes the need to convert handlers at every route.
type Middleware = func(http.Handler) http.Handler
