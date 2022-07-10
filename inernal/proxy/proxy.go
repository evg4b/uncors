package proxy

import (
	"net/http"

	"github.com/evg4b/uncors/inernal/infrastrucure"
)

type ProxyMiddleware struct {
}

func NewProxyHandlingMiddleware() *ProxyMiddleware {
	return &ProxyMiddleware{}
}

func (pm *ProxyMiddleware) Wrap(next infrastrucure.HandlerFunc) infrastrucure.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		next(w, r)
	}
}
