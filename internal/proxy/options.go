package proxy

import (
	"net/http"

	"github.com/evg4b/uncors/internal/urlreplacer"
)

type proxyMiddlewareOption = func(*ProxyMiddleware)

func WithURLReplacerFactory(replacerFactory *urlreplacer.URLReplacerFactory) proxyMiddlewareOption {
	return func(pm *ProxyMiddleware) {
		pm.replacerFactory = replacerFactory
	}
}

func WithHTTPClient(http http.Client) proxyMiddlewareOption {
	return func(pm *ProxyMiddleware) {
		pm.http = http
	}
}
