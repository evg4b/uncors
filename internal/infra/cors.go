package infra

import (
	"net/http"

	"github.com/go-http-utils/headers"
)

const (
	allowMethods = "GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, HEAD, LINK, OPTIONS"
	maxAge       = "86400"
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

var mapping = map[string]string{
	headers.AccessControlAllowOrigin:  headers.Origin,
	headers.AccessControlAllowHeaders: headers.AccessControlRequestHeaders,
	headers.AccessControlAllowMethods: headers.AccessControlRequestMethod,
}

var defaultValues = map[string]string{
	headers.AccessControlAllowOrigin:  "*",
	headers.AccessControlAllowHeaders: "*",
	headers.AccessControlAllowMethods: allowMethods,
}

func WriteCorsHeadersForOptions(respHeader, reqHeader http.Header) {
	for respKey, reqKey := range mapping {
		if value := reqHeader.Get(reqKey); value != "" {
			respHeader.Set(respKey, value)
		} else {
			respHeader.Set(respKey, defaultValues[respKey])
		}
	}

	respHeader.Set(headers.AccessControlAllowCredentials, "true")
	respHeader.Set(headers.AccessControlMaxAge, maxAge)
	respHeader.Set(headers.AccessControlExposeHeaders, "*")
}
