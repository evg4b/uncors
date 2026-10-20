package server

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

// Headers recorded on the request span: the ones that explain CORS behavior
// or identify the request. Credentials and cookies are never recorded.
var (
	tracedRequestHeaders = tracedHeaders("http.request.header.",
		"Origin",
		"Referer",
		"Content-Type",
		"Access-Control-Request-Method",
		"Access-Control-Request-Headers",
		"X-Request-Id",
		"X-Correlation-Id",
	)
	tracedResponseHeaders = tracedHeaders("http.response.header.",
		"Content-Type",
		"Location",
		"Cache-Control",
		"Access-Control-Allow-Origin",
		"Access-Control-Allow-Credentials",
		"Access-Control-Allow-Methods",
		"Access-Control-Allow-Headers",
	)
)

// Upper bounds of the attributes annotateRequest and annotateResponse add,
// so their slices are allocated once.
const (
	requestAttributes  = 12
	responseAttributes = 3
)

// tracedHeader pairs a canonical header name with its semconv attribute key,
// built once rather than for every request.
type tracedHeader struct {
	name string
	key  attribute.Key
}

func tracedHeaders(prefix string, names ...string) []tracedHeader {
	headers := make([]tracedHeader, 0, len(names))
	for _, name := range names {
		headers = append(headers, tracedHeader{
			name: http.CanonicalHeaderKey(name),
			key:  attribute.Key(prefix + strings.ToLower(name)),
		})
	}

	return headers
}

func appendHeaders(attrs []attribute.KeyValue, traced []tracedHeader, header http.Header) []attribute.KeyValue {
	for _, tracedHeader := range traced {
		if values := header[tracedHeader.name]; len(values) > 0 {
			attrs = append(attrs, tracedHeader.key.StringSlice(values))
		}
	}

	return attrs
}

func annotateRequest(span trace.Span, request *http.Request, requestID uint64) {
	host, port := splitHostPort(request.Host)
	attrs := make([]attribute.KeyValue, 0, requestAttributes+len(tracedRequestHeaders))
	attrs = append(attrs,
		attribute.Int64("uncors.request.id", int64(requestID&math.MaxInt64)),
		semconv.HTTPRequestMethodKey.String(request.Method),
		semconv.URLFull(helpers.LogURL(request.URL)),
		semconv.URLScheme(request.URL.Scheme),
		semconv.URLPath(request.URL.Path),
		semconv.ServerAddress(host),
		semconv.NetworkProtocolVersion(telemetry.ProtocolVersion(request.ProtoMajor, request.ProtoMinor)),
	)

	if port > 0 {
		attrs = append(attrs, semconv.ServerPort(port))
	}

	if clientHost, clientPort := splitHostPort(request.RemoteAddr); clientHost != "" {
		attrs = append(attrs, semconv.ClientAddress(clientHost), semconv.ClientPort(clientPort))
	}

	if userAgent := request.UserAgent(); userAgent != "" {
		attrs = append(attrs, semconv.UserAgentOriginal(userAgent))
	}

	if request.ContentLength >= 0 {
		attrs = append(attrs, semconv.HTTPRequestBodySize(int(request.ContentLength)))
	}

	attrs = appendHeaders(attrs, tracedRequestHeaders, request.Header)

	span.SetName("HTTP " + request.Method)
	span.SetAttributes(attrs...)
}

func annotateResponse(span trace.Span, data *contracts.RequestData, header http.Header, written int64) {
	attrs := make([]attribute.KeyValue, 0, responseAttributes+len(tracedResponseHeaders))
	attrs = append(attrs,
		semconv.HTTPResponseStatusCode(data.Code),
		semconv.HTTPResponseBodySize(int(written)),
		attribute.Bool("uncors.cancelled", data.Cancelled),
	)
	attrs = appendHeaders(attrs, tracedResponseHeaders, header)

	span.SetAttributes(attrs...)
}

func splitHostPort(address string) (string, int) {
	host, rawPort, err := net.SplitHostPort(address)
	if err != nil {
		return address, 0
	}

	port, _ := strconv.Atoi(rawPort)

	return host, port
}
