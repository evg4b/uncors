package telemetry

import (
	"context"
	"net/http"
	"net/http/httptrace"

	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/propagation"
)

// TraceIDHeader carries the request's trace ID in responses while tracing is
// on, so a request seen in the browser can be found in the trace viewer.
const TraceIDHeader = "X-Uncors-Trace-Id"

// WrapTransport traces the requests sent through transport: a client span per
// request, with DNS, connection, and TLS sub-spans and the request headers
// (credentials and cookies redacted), and the OTel HTTP client metrics. When
// tracing is off it returns transport unchanged.
//
// Trace context is never injected, so upstream servers see the request as
// the client sent it.
func WrapTransport(transport http.RoundTripper) http.RoundTripper {
	state := current.Load()
	if state == nil {
		return transport
	}

	tracerProvider := state.providers.Tracer

	return otelhttp.NewTransport(transport,
		otelhttp.WithTracerProvider(tracerProvider),
		otelhttp.WithMeterProvider(state.providers.Meter),
		otelhttp.WithPropagators(propagation.NewCompositeTextMapPropagator()),
		otelhttp.WithSpanNameFormatter(func(_ string, request *http.Request) string {
			return "HTTP " + request.Method
		}),
		otelhttp.WithClientTrace(func(ctx context.Context) *httptrace.ClientTrace {
			return otelhttptrace.NewClientTrace(ctx, otelhttptrace.WithTracerProvider(tracerProvider))
		}),
	)
}
