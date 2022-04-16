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

	req, err := http.NewRequest(req.Method, url, req.Body)
	if err != nil {
		log.Print(err)
		return
	}

	for n, h := range req.Header {
		for _, h := range h {
			req.Header.Add(n, h)
		}
	}

	client := http.Client{}
	if req.TLS != nil {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	for h, v := range resp.Header {
		for _, v := range v {
			w.Header().Add(h, v)
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
