package router

import (
	"maps"
	"slices"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

// Kinds of route, recorded as uncors.route.kind.
const (
	routeStatic  = "static"
	routeMock    = "mock"
	routeScript  = "script"
	routeRewrite = "rewrite"
	routeDefault = "default"
)

// traced records which mapping and route matched on the request's root
// span. The attributes are built once, with the router; when tracing is off
// the handler is returned as is.
func traced(handler contracts.Handler, attrs ...attribute.KeyValue) contracts.Handler {
	if !telemetry.Enabled() {
		return handler
	}

	return infra.HandlerFunc(func(writer contracts.ResponseWriter, request *contracts.Request) error {
		if span := trace.SpanFromContext(request.Context()); span.IsRecording() {
			span.SetAttributes(attrs...)
		}

		return handler.ServeHTTP(writer, request)
	})
}

func mappingAttributes(mapping config.Mapping, kind string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("uncors.mapping.from", mapping.From.String()),
		attribute.String("uncors.mapping.to", mapping.To.String()),
		attribute.String("uncors.route.kind", kind),
	}
}

func matcherAttributes(matcher config.RequestMatcher, index int) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		semconv.HTTPRoute(matcher.Path),
		attribute.Int("uncors.route.index", index),
	}

	if matcher.Method != "" {
		attrs = append(attrs, attribute.String("uncors.route.method", matcher.Method))
	}

	if len(matcher.Queries) > 0 {
		attrs = append(attrs, attribute.StringSlice("uncors.route.queries", pairs(matcher.Queries)))
	}

	if len(matcher.Headers) > 0 {
		attrs = append(attrs, attribute.StringSlice("uncors.route.headers", pairs(matcher.Headers)))
	}

	return attrs
}

func pairs(values map[string]string) []string {
	keys := slices.Sorted(maps.Keys(values))
	result := make([]string, 0, len(keys))

	for _, key := range keys {
		result = append(result, key+"="+values[key])
	}

	return result
}
