package infrastructure

import "net/http"

func NormalizeHTTPReqDecorator(handler baseHandlerFunc) baseHandlerFunc {
	return normalizeReqDecorator("http", handler)
}

func NormalizeHTTPSReqDecorator(handler baseHandlerFunc) baseHandlerFunc {
	return normalizeReqDecorator("https", handler)
}

func normalizeReqDecorator(protocol string, handler baseHandlerFunc) baseHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.URL.Host = r.Host
		r.URL.Scheme = protocol
		handler(w, r)
	}
}
