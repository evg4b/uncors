package proxy

import (
	"net/http"

	"github.com/evg4b/uncors/internal/urlreplacer"
)

type Replcaer interface {
	ToTarget(targetUrl string) (string, error)
	ToSource(targetUrl string, host string) (string, error)
}

type proxyMiddlewareOptions = func(*ProxyMiddleware)

func WithUrlReplacerFactory(replacerFactory *urlreplacer.UrlReplacerFactory) proxyMiddlewareOptions {
	return func(pm *ProxyMiddleware) {
		pm.replacerFactory = replacerFactory
	}
}

func WithHttpClient(http http.Client) proxyMiddlewareOptions {
	return func(pm *ProxyMiddleware) {
		pm.http = http
	}
}
