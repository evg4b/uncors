package proxy

import (
	"net/http"
	"net/url"

	"github.com/evg4b/uncors/internal/urlreplacer"
)

type URLReplacerFactory interface {
	Make(requestURL *url.URL) (*urlreplacer.Replacer, error)
	MakeV2(requestURL *url.URL) (*urlreplacer.ReplacerV2, *urlreplacer.ReplacerV2, error)
}

type MiddlewareOption = func(*ProxyMiddleware)

func WithURLReplacerFactory(replacerFactory URLReplacerFactory) MiddlewareOption {
	return func(pm *ProxyMiddleware) {
		pm.replacerFactory = replacerFactory
	}
}

func WithHTTPClient(http *http.Client) MiddlewareOption {
	return func(pm *ProxyMiddleware) {
		pm.http = http
	}
}
