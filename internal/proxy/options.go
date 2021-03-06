package proxy

import (
	"net/http"
	"net/url"

	"github.com/evg4b/uncors/internal/urlreplacer"
)

type URLReplacerFactory interface {
	Make(requestURL *url.URL) (*urlreplacer.Replacer, error)
}

type proxyMiddlewareOption = func(*ProxyMiddleware)

func WithURLReplacerFactory(replacerFactory URLReplacerFactory) proxyMiddlewareOption {
	return func(pm *ProxyMiddleware) {
		pm.replacerFactory = replacerFactory
	}
}

func WithHTTPClient(http *http.Client) proxyMiddlewareOption {
	return func(pm *ProxyMiddleware) {
		pm.http = http
	}
}
