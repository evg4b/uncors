package proxy

import (
	"io"
	"net/http"

	"github.com/evg4b/uncors/internal/infrastrucure"
	"github.com/pterm/pterm"
)

type ProxyMiddleware struct {
	replcaer Replcaer
	http     http.Client
}

func NewProxyHandlingMiddleware(options ...proxyMiddlewareOptions) *ProxyMiddleware {
	middleware := &ProxyMiddleware{}

	for _, option := range options {
		option(middleware)
	}

	return middleware
}

func (pm *ProxyMiddleware) Wrap(next infrastrucure.HandlerFunc) infrastrucure.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) error {
		if req.Method == "OPTIONS" {
			return pm.hadnleOptionsRequest(w, req)
		}

		url, err := pm.replcaer.ToTarget(req.URL.String())
		if err != nil {
			return err
		}

		originRequet, err := http.NewRequest(req.Method, url, req.Body)
		if err != nil {
			pterm.Error.Println(err)

			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return err
		}

		err = copyHeaders(req.Header, originRequet.Header, map[string]func(string) (string, error){
			"origin":  pm.targetUrlReplace,
			"referer": pm.targetUrlReplace,
		})

		if err != nil {
			return err
		}

		for _, cookie := range req.Cookies() {
			cookie.Secure = true
			originRequet.AddCookie(cookie)
		}

		resp, err := pm.http.Do(originRequet)
		if err != nil {
			return err
		}

		for _, cookie := range resp.Cookies() {
			cookie.Secure = false
			http.SetCookie(w, cookie)
		}

		header := w.Header()
		err = copyHeaders(resp.Header, header, map[string]func(string) (string, error){
			"location": func(url string) (string, error) {
				return pm.replcaer.ToSource(url, req.URL.Hostname())
			},
		})

		if err != nil {
			return err
		}

		header.Set("Access-Control-Allow-Origin", "*")
		header.Set("Access-Control-Allow-Credentials", "true")
		header.Set("Access-Control-Allow-Methods", "GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, HEAD, LINK, OPTIONS")

		w.WriteHeader(resp.StatusCode)

		_, err = io.Copy(w, resp.Body)
		if err != nil {
			return err
		}

		pterm.Success.Println(url)

		return nil
	}
}

func (pm *ProxyMiddleware) targetUrlReplace(url string) (string, error) {
	return pm.replcaer.ToTarget(url)
}
