package handler

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

type RequestHandeler struct {
	target   string
	protocol string
	origin   string
	origin2  string
}

func NewRequestHandler(options ...RequestHandelerOptions) *RequestHandeler {
	handler := &RequestHandeler{}

	for _, option := range options {
		option(handler)
	}

	return handler
}

func (rh *RequestHandeler) HandleRequest(w http.ResponseWriter, req *http.Request) {
	url := fmt.Sprintf("%s://%s%s", rh.protocol, rh.target, req.URL.String())
	log.Println("Requset: ", url)

	for n, h := range req.Header {
		if strings.Contains(n, "Origin") {
			for _, h := range h {
				rh.origin = h
			}
		}
	}
	header := w.Header()

	header.Add("Access-Control-Allow-Origin", rh.origin)
	header.Add("Access-Control-Allow-Credentials", "true")
	header.Add("Access-Control-Allow-Methods", "GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, HEAD, LINK, OPTIONS")

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
				req2.Header.Add(n, h)
			}
		}
	}
	for _, cookie := range req.Cookies() {
		cookie.Secure = true
		req2.AddCookie(cookie)
	}

	req2.Header.Set("origin", "https://github.com")
	req2.Header.Set("referer", "https://github.com/")

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
	req2.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 6.3; Trident/7.0; rv:11.0) like Gecko")
	resp, err := client.Do(req2)
	if err != nil {
		log.Println(err)
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
				if strings.ToLower(h) == "location" 					v = strings.ReplaceAll(v, rh.target, rh.origin2)
					v = strings.ReplaceAll(v, "https", "http")
				}
				w.Header().Add(h, v)
			}
		}
	}

	w.WriteHeader(resp.StatusCode)

	wr, err := io.Copy(w, resp.Body)
	if err != nil {
		log.Println(wr, err)
	} else {
		log.Print("Written", wr, "bytes")
	}
}
