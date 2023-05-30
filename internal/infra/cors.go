package infra

import (
	"net/http"

	"github.com/go-http-utils/headers"
)

const allowMethods = "GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, HEAD, LINK, OPTIONS"

func WriteCorsHeaders(header http.Header) {
	header.Set(headers.AccessControlAllowOrigin, "*")
	header.Set(headers.AccessControlAllowCredentials, "true")
	header.Set(headers.AccessControlAllowMethods, allowMethods)
}
