package proxy

import (
	"crypto/tls"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/evg4b/uncors/internal/infrastrucure"
	"github.com/pterm/pterm"
)

type ProxyMiddleware struct {
	replcaer Replcaer
}

func NewProxyHandlingMiddleware(options ...ProxyMiddlewareOptions) *ProxyMiddleware {
	middleware := &ProxyMiddleware{}

	for _, option := range options {
		option(middleware)
	}

	return middleware
}

func (pm *ProxyMiddleware) Wrap(next infrastrucure.HandlerFunc) infrastrucure.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		url, _ := pm.replcaer.ToTarget(req.URL.String())

		header := w.Header()

		header.Set("Access-Control-Allow-Origin", "*")
		header.Set("Access-Control-Allow-Credentials", "true")
		header.Set("Access-Control-Allow-Methods", "GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, HEAD, LINK, OPTIONS")

		if req.Method == "OPTIONS" {
			log.Print("CORS asked for ", url)
			for n, h := range req.Header {
				if strings.Contains(n, "Access-Control-Request") {
					for _, h := range h {
						k := strings.Replace(n, "Request", "Allow", 1)
						header.Add(k, h)
					}
				}
			}
			return
		}

		req2, err := http.NewRequest(req.Method, url, req.Body)
		if err != nil {
			log.Print(err)
			return
		}

		for n, h := range req.Header {
			if strings.ToLower(n) != "cookie" {
				for _, h := range h {
					if strings.ToLower(n) == "origin" || strings.ToLower(n) == "referer" {
						h, err = pm.replcaer.ToTarget(h)
						if err != nil {
							panic(err)
						}
					}

					req2.Header.Add(n, h)
				}
			}
		}
		for _, cookie := range req.Cookies() {
			cookie.Secure = true
			req2.AddCookie(cookie)
		}

		client := http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		if req2.TLS != nil {
			client.Transport = &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
		}

		resp, err := client.Do(req2)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		for _, cookie := range resp.Cookies() {
			cookie.Secure = false
			http.SetCookie(w, cookie)
		}

		for h, v2 := range resp.Header {
			if strings.ToLower(h) != "set-cookie" {
				for _, v := range v2 {
					if strings.ToLower(h) == "location" {
						v, err = pm.replcaer.ToSource(v, req.Host)
						if err != nil {
							panic(err)
						}
					}
					w.Header().Add(h, v)
				}
			}
		}

		w.WriteHeader(resp.StatusCode)

		_, err = io.Copy(w, resp.Body)
		if err != nil {
			log.Println(err)
			return
		}

		pterm.Success.Println(url)
	}
}
