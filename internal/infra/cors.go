package infra

import (
	"net/http"
	"slices"
	"strings"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/go-http-utils/headers"
)

const (
	allowMethods = "GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS"
	maxAge       = "86400"
	anyValue     = "*"

	corsHeaderPrefix       = "Access-Control-"
	exposedHeadersCapacity = 32
)

func WriteCorsHeaders(header http.Header, origin string) {
	writeCorsHeaders(header, origin)
	writeExposeHeaders(header, origin)
}

func writeCorsHeaders(header http.Header, origin string) {
	if origin != "" {
		header.Set(headers.AccessControlAllowOrigin, origin)
		addVary(header, headers.Origin)
	} else {
		header.Set(headers.AccessControlAllowOrigin, anyValue)
	}

	header.Set(headers.AccessControlAllowCredentials, "true")
	header.Set(headers.AccessControlAllowMethods, allowMethods)
	header.Set(headers.AccessControlMaxAge, maxAge)
	header.Del(headers.AccessControlAllowHeaders)
	header.Del(headers.AccessControlExposeHeaders)
}

func writeExposeHeaders(header http.Header, origin string) {
	if origin == "" {
		header.Set(headers.AccessControlExposeHeaders, anyValue)

		return
	}

	if value := exposedHeaders(header); value != "" {
		header.Set(headers.AccessControlExposeHeaders, value)
	} else {
		header.Del(headers.AccessControlExposeHeaders)
	}
}

func exposedHeaders(header http.Header) string {
	names := make([]string, 0, exposedHeadersCapacity)
	size := 0

	for name := range header {
		if isExposedHeader(name) {
			names = append(names, name)
			size += len(name) + len(", ")
		}
	}

	if len(names) == 0 {
		return ""
	}

	slices.Sort(names)

	var builder strings.Builder

	builder.Grow(size)

	for index, name := range names {
		if index > 0 {
			builder.WriteString(", ")
		}

		builder.WriteString(name)
	}

	return builder.String()
}

func isExposedHeader(name string) bool {
	switch name {
	case headers.CacheControl,
		headers.ContentLanguage,
		headers.ContentLength,
		headers.ContentType,
		headers.Expires,
		headers.LastModified,
		headers.Pragma,
		headers.SetCookie,
		"Set-Cookie2":
		return false
	}

	return len(name) < len(corsHeaderPrefix) || !strings.EqualFold(name[:len(corsHeaderPrefix)], corsHeaderPrefix)
}

func addVary(header http.Header, name string) {
	for _, value := range header[headers.Vary] {
		for token := range strings.SplitSeq(value, ",") {
			token = strings.TrimSpace(token)
			if token == anyValue || strings.EqualFold(token, name) {
				return
			}
		}
	}

	header.Add(headers.Vary, name)
}

type preflightHeader struct {
	response     string
	request      string
	defaultValue string
}

var preflightHeaders = []preflightHeader{
	{headers.AccessControlAllowOrigin, headers.Origin, anyValue},
	{headers.AccessControlAllowMethods, headers.AccessControlRequestMethod, allowMethods},
	{headers.AccessControlAllowHeaders, headers.AccessControlRequestHeaders, anyValue},
}

func WriteCorsHeadersForOptions(respHeader, reqHeader http.Header) {
	for _, item := range preflightHeaders {
		if value := reqHeader.Get(item.request); value != "" {
			respHeader.Set(item.response, value)
			addVary(respHeader, item.request)
		} else {
			respHeader.Set(item.response, item.defaultValue)
		}
	}

	respHeader.Set(headers.AccessControlAllowCredentials, "true")
	respHeader.Set(headers.AccessControlMaxAge, maxAge)
	respHeader.Set(headers.AccessControlExposeHeaders, anyValue)
}

type CorsWriter struct {
	contracts.ResponseWriter

	origin  string
	exposed bool
}

func NewCorsWriter(writer contracts.ResponseWriter, origin string) *CorsWriter {
	writeCorsHeaders(writer.Header(), origin)

	return &CorsWriter{ResponseWriter: writer, origin: origin}
}

func (w *CorsWriter) WriteHeader(statusCode int) {
	w.ExposeHeaders()
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *CorsWriter) Write(data []byte) (int, error) {
	w.ExposeHeaders()

	return w.ResponseWriter.Write(data)
}

func (w *CorsWriter) ExposeHeaders() {
	if w.exposed {
		return
	}

	w.exposed = true

	header := w.Header()
	if len(header[headers.AccessControlExposeHeaders]) == 0 {
		writeExposeHeaders(header, w.origin)
	}
}

func (w *CorsWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
