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

func Is2xxCode(code int) bool {
	return 200 <= code && code < 300
}

func BodyAllowed(code int) bool {
	return code >= http.StatusOK && code != http.StatusNoContent && code != http.StatusNotModified
}

func NormaliseStatusCode(code int) int {
	if code == 0 {
		return http.StatusOK
	}

	return code
}
