package infra

import (
	"net/http"

	"github.com/go-http-utils/headers"
)

const (
	allowMethods = "GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, HEAD, LINK, OPTIONS"
	maxAge       = "86400" // 24 hours in seconds
)

// SetHeaderOrDefault sets a header to the provided value if not empty, otherwise sets it to the default value.
func SetHeaderOrDefault(header http.Header, key, value, defaultValue string) {
	if value != "" {
		header.Set(key, value)
	} else {
		header.Set(key, defaultValue)
	}
}

func WriteCorsHeaders(header http.Header, origin string) {
	if origin != "" {
		header.Set(headers.AccessControlAllowOrigin, origin)
	} else {
		header.Set(headers.AccessControlAllowOrigin, "*")
	}

	header.Set(headers.AccessControlAllowCredentials, "true")
	header.Set(headers.AccessControlAllowHeaders, "*")
	header.Set(headers.AccessControlAllowMethods, allowMethods)
	header.Set(headers.AccessControlMaxAge, maxAge)
	header.Set(headers.AccessControlExposeHeaders, "*")
}

// WriteCorsHeadersForOptions writes CORS headers for OPTIONS preflight requests.
// It uses request headers to set appropriate CORS response headers with fallback defaults.
func WriteCorsHeadersForOptions(respHeader http.Header, reqHeader http.Header) {
	SetHeaderOrDefault(respHeader, headers.AccessControlAllowOrigin, reqHeader.Get(headers.Origin), "*")
	respHeader.Set(headers.AccessControlAllowCredentials, "true")
	SetHeaderOrDefault(respHeader, headers.AccessControlAllowHeaders, reqHeader.Get(headers.AccessControlRequestHeaders), "*")
	SetHeaderOrDefault(respHeader, headers.AccessControlAllowMethods, reqHeader.Get(headers.AccessControlRequestMethod), allowMethods)
	respHeader.Set(headers.AccessControlMaxAge, maxAge)
	respHeader.Set(headers.AccessControlExposeHeaders, "*")
}
