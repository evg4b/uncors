package helpers

import "net/http"

func NormaliseRequest(request *http.Request) {
	request.URL.Host = request.Host

	if request.TLS != nil {
		request.URL.Scheme = "https"
	} else {
		request.URL.Scheme = "http"
	}
}

func Is1xxCode(code int) bool {
	return 100 <= code && code < 200
}

func Is2xxCode(code int) bool {
	return 200 <= code && code < 300
}

func Is3xxCode(code int) bool {
	return 300 <= code && code < 400
}

func Is4xxCode(code int) bool {
	return 400 <= code && code < 500
}

func Is5xxCode(code int) bool {
	return 500 <= code && code < 600
}
