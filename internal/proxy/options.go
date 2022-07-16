package proxy

import (
	"net/http"

	"github.com/evg4b/uncors/internal/urlreplacer"
)

type proxyMiddlewareOption = func(*ProxyMiddleware)

func WithUrlReplacerFactory(replacerFactory *urlreplacer.UrlReplacerFactory) proxyMiddlewareOption {
	return func(pm *ProxyMiddleware) {
		pm.replacerFactory = replacerFactory
	}
}

func WithHttpClient(http http.Client) proxyMiddlewareOption {
	return func(pm *ProxyMiddleware) {
		pm.http = http
	}
}
