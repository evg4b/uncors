package infra

import (
	"net/http"

	"github.com/go-http-utils/headers"
)

const (
	allowMethods = "GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, LINK, OPTIONS"
	maxAge       = "86400" // 24 hours in seconds
)

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
