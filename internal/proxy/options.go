package proxy

type Replcaer interface {
	ToTarget(targetUrl string) (string, error)
	ToSource(targetUrl string, host string) (string, error)
}

type ProxyMiddlewareOptions = func(*ProxyMiddleware)

func WithUrlReplcaer(replcaer Replcaer) ProxyMiddlewareOptions {
	return func(pm *ProxyMiddleware) {
		pm.replcaer = replcaer
	}
}
