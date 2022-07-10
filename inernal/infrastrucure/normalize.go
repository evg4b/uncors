package infrastrucure

import "net/http"

func NormalizeHttpReqDecorator(handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return normalizeReqDecorator("http", handler)
}

func NormalizeHttpsReqDecorator(handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return normalizeReqDecorator("https", handler)
}

func normalizeReqDecorator(protocol string, handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		r.URL.Host = r.Host
		r.URL.Scheme = protocol
	}
}
