package infra

import (
	"net/http"
	"strings"

	"github.com/evg4b/uncors/internal/contracts"
)

// NormaliseRequest fills in the scheme and host that a server side request
// carries out of band, so that the rest of the pipeline can work with a complete
// URL.
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

// NormaliseStatusCode reports the status a response with no explicit WriteHeader
// actually has.
func NormaliseStatusCode(code int) int {
	if code == 0 {
		return http.StatusOK
	}

	return code
}

// ToRequestData summarises a request for the activity log.
func ToRequestData(req *http.Request, code int) *contracts.RequestData {
	return &contracts.RequestData{
		Method: req.Method,
		URL:    req.URL,
		Header: req.Header,
		Body:   nil,
		Code:   code,
	}
}

// SanitizeLogValue neutralises user-controlled input before it is written to a
// log entry, preventing log-forging via embedded line breaks. Carriage returns
// and line feeds are replaced with spaces so a single logical entry stays on a
// single line.
func SanitizeLogValue(value string) string {
	return strings.NewReplacer("\r", " ", "\n", " ").Replace(value)
}
