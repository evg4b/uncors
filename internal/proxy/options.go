package proxy

import "net/http"

type Replcaer interface {
	ToTarget(targetUrl string) (string, error)
	ToSource(targetUrl string, host string) (string, error)
}

type proxyMiddlewareOptions = func(*ProxyMiddleware)

func WithUrlReplcaer(replcaer Replcaer) proxyMiddlewareOptions {
	return func(pm *ProxyMiddleware) {
		pm.replcaer = replcaer
	}
}

func WithHttpClient(http http.Client) proxyMiddlewareOptions {
	return func(pm *ProxyMiddleware) {
		pm.http = http
	}
}
