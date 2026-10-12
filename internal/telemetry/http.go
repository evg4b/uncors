package telemetry

import (
	"context"
	"net/http"
	"net/http/httptrace"
	"strconv"

	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/propagation"
)

// TraceIDHeader carries the request's trace ID in responses while tracing is
// on, so a request seen in the browser can be found in the trace viewer.
const TraceIDHeader = "X-Uncors-Trace-Id"

// ProtocolVersion is the network.protocol.version of an HTTP message, built
// without allocating for the usual versions.
func ProtocolVersion(major, minor int) string {
	switch {
	case major == 1 && minor == 1:
		return "1.1"
	case major == 2 && minor == 0:
		return "2"
	case major == 1 && minor == 0:
		return "1.0"
	case minor == 0 && major > 1:
		// Semconv drops the minor version from HTTP/2 on.
		return strconv.Itoa(major)
	default:
		return strconv.Itoa(major) + "." + strconv.Itoa(minor)
	}
}

// WrapTransport traces the requests sent through transport: a client span per
// request, with its headers (credentials and cookies redacted), and the OTel
// HTTP client metrics. With TraceConnections the span also gets events for
// the DNS, connection, and TLS phases. When tracing is off it returns
// transport unchanged.
//
// Trace context is never injected, so upstream servers see the request as
// the client sent it.
func WrapTransport(transport http.RoundTripper) http.RoundTripper {
	state := current.Load()
	if state == nil {
		return transport
	}

	tracerProvider := state.providers.Tracer
	options := []otelhttp.Option{
		otelhttp.WithTracerProvider(tracerProvider),
		otelhttp.WithMeterProvider(state.providers.Meter),
		otelhttp.WithPropagators(propagation.NewCompositeTextMapPropagator()),
		otelhttp.WithSpanNameFormatter(func(_ string, request *http.Request) string {
			return "HTTP " + request.Method
		}),
	}

	// The phases are events rather than sub-spans, which would cost several
	// spans per request. Even so they nearly double the allocations of an
	// upstream request, so they are opt-in.
	if state.providers.TraceConnections {
		options = append(options, otelhttp.WithClientTrace(func(ctx context.Context) *httptrace.ClientTrace {
			return otelhttptrace.NewClientTrace(ctx,
				otelhttptrace.WithTracerProvider(tracerProvider),
				otelhttptrace.WithoutSubSpans(),
			)
		}))
	}

	return otelhttp.NewTransport(transport, options...)
}
