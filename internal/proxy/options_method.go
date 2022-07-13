package proxy

import (
	"net/http"
	"strings"
)

func (pm *ProxyMiddleware) hadnleOptionsRequest(w http.ResponseWriter, req *http.Request) error {
	header := w.Header()
	for n, h := range req.Header {
		if strings.Contains(n, "Access-Control-Request") {
			for _, h := range h {
				k := strings.Replace(n, "Request", "Allow", 1)
				header.Add(k, h)
			}
		}
	}

	return nil
}
