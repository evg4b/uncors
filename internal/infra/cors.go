package infra

import (
	"net/http"
	"slices"
	"strings"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/tracking"
	"github.com/go-http-utils/headers"
)

const (
	allowMethods = "GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS"
	maxAge       = "86400"
	anyValue     = "*"
	dateHeader   = "Date"

	corsHeaderPrefix       = "Access-Control-"
	exposedHeadersCapacity = 32
)

func WriteRequestCorsHeaders(header http.Header, request *http.Request) {
	if tracking.OriginRejected(request.Context()) {
		WithholdCorsHeaders(header)

		return
	}

	WriteCorsHeaders(header, request.Header.Get(headers.Origin))
}

func WithholdCorsHeaders(header http.Header) {
	for name := range header {
		if isCorsHeader(name) {
			delete(header, name)
		}
	}

	addVary(header, headers.Origin)
}

func WriteCorsHeaders(header http.Header, origin string) {
	writeCorsHeaders(header, origin)
	writeExposeHeaders(header, origin)
}

func writeCorsHeaders(header http.Header, origin string) {
	if origin != "" {
		addVary(header, headers.Origin)
	} else {
		origin = anyValue
	}

	values := []string{origin, "true", allowMethods, maxAge}
	header[headers.AccessControlAllowOrigin] = values[0:1:1]
	header[headers.AccessControlAllowCredentials] = values[1:2:2]
	header[headers.AccessControlAllowMethods] = values[2:3:3]
	header[headers.AccessControlMaxAge] = values[3:4:4]
	header.Del(headers.AccessControlAllowHeaders)
	header.Del(headers.AccessControlExposeHeaders)
}

func writeExposeHeaders(header http.Header, origin string) {
	if origin == "" {
		header.Set(headers.AccessControlExposeHeaders, anyValue)

		return
	}

	header.Set(headers.AccessControlExposeHeaders, exposedHeaders(header))
}

func exposedHeaders(header http.Header) string {
	names := append(make([]string, 0, exposedHeadersCapacity), dateHeader)
	size := len(dateHeader)

	for name := range header {
		if name != dateHeader && isExposedHeader(name) {
			names = append(names, name)
			size += len(name) + len(", ")
		}
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

	return !isCorsHeader(name)
}

func isCorsHeader(name string) bool {
	return len(name) >= len(corsHeaderPrefix) && strings.EqualFold(name[:len(corsHeaderPrefix)], corsHeaderPrefix)
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

	origin   string
	rejected bool
	exposed  bool
}

func NewCorsWriter(writer contracts.ResponseWriter, request *http.Request) *CorsWriter {
	corsWriter := &CorsWriter{
		ResponseWriter: writer,
		origin:         request.Header.Get(headers.Origin),
		rejected:       tracking.OriginRejected(request.Context()),
	}

	if !corsWriter.rejected {
		writeCorsHeaders(writer.Header(), corsWriter.origin)
	}

	return corsWriter
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
	if w.rejected {
		WithholdCorsHeaders(header)

		return
	}

	if len(header[headers.AccessControlExposeHeaders]) == 0 {
		writeExposeHeaders(header, w.origin)
	}
}

func (w *CorsWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
