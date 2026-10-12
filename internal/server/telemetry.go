package server

import (
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

// Headers recorded on the request span: the ones that explain CORS behavior
// or identify the request. Credentials and cookies are never recorded.
var (
	tracedRequestHeaders = []string{
		"Origin",
		"Referer",
		"Content-Type",
		"Access-Control-Request-Method",
		"Access-Control-Request-Headers",
		"X-Request-Id",
		"X-Correlation-Id",
	}
	tracedResponseHeaders = []string{
		"Content-Type",
		"Location",
		"Cache-Control",
		"Access-Control-Allow-Origin",
		"Access-Control-Allow-Credentials",
		"Access-Control-Allow-Methods",
		"Access-Control-Allow-Headers",
	}
)

func annotateRequest(span trace.Span, request *http.Request, requestID uint64) {
	host, port := splitHostPort(request.Host)
	attrs := []attribute.KeyValue{
		attribute.Int64("uncors.request.id", int64(requestID)), //nolint:gosec // request ids never reach MaxInt64
		semconv.HTTPRequestMethodKey.String(request.Method),
		semconv.URLFull(helpers.LogURL(request.URL)),
		semconv.URLScheme(request.URL.Scheme),
		semconv.URLPath(request.URL.Path),
		semconv.ServerAddress(host),
		semconv.NetworkProtocolVersion(strconv.Itoa(request.ProtoMajor) + "." + strconv.Itoa(request.ProtoMinor)),
	}

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

	for _, name := range tracedRequestHeaders {
		if values := request.Header.Values(name); len(values) > 0 {
			attrs = append(attrs, semconv.HTTPRequestHeader(strings.ToLower(name), values...))
		}
	}

	span.SetName("HTTP " + request.Method)
	span.SetAttributes(attrs...)
}

func annotateResponse(span trace.Span, data *contracts.RequestData, header http.Header, written int64) {
	attrs := []attribute.KeyValue{
		semconv.HTTPResponseStatusCode(data.Code),
		semconv.HTTPResponseBodySize(int(written)),
		attribute.Bool("uncors.cancelled", data.Cancelled),
	}

	for _, name := range tracedResponseHeaders {
		if values := header.Values(name); len(values) > 0 {
			attrs = append(attrs, semconv.HTTPResponseHeader(strings.ToLower(name), values...))
		}
	}

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
