package router_test

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/telemetry"
	"github.com/evg4b/uncors/internal/tracking"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

const rootSpanName = "request"

// serveTraced serves one request under a root span, as the server does, and
// returns the ended spans of its trace.
func serveTraced(
	t *testing.T,
	spans *tracetest.SpanRecorder,
	handler http.Handler,
	method, url string,
) []sdktrace.ReadOnlySpan {
	t.Helper()

	request := httptest.NewRequestWithContext(t.Context(), method, url, nil)
	helpers.NormaliseRequest(request)

	ctx, root := telemetry.StartServer(request.Context(), rootSpanName)
	ctx, scope := tracking.Begin(ctx, events.NoopPublisher{}, 1, request)
	recorder := httptest.NewRecorder()

	serveHTTP(t, handler, recorder, request.WithContext(ctx))
	scope.Finish(helpers.ToRequestData(request, recorder.Code))
	root.End()

	traceID := root.SpanContext().TraceID()

	return slices.DeleteFunc(spans.Ended(), func(span sdktrace.ReadOnlySpan) bool {
		return span.SpanContext().TraceID() != traceID
	})
}

// spanTree lists the "parent > child" edges of the trace, by span name.
func spanTree(ended []sdktrace.ReadOnlySpan) []string {
	names := map[string]string{}
	for _, span := range ended {
		names[span.SpanContext().SpanID().String()] = span.Name()
	}

	var edges []string

	for _, span := range ended {
		if parent, ok := names[span.Parent().SpanID().String()]; ok {
			edges = append(edges, parent+" > "+span.Name())
		}
	}

	return edges
}

func rootSpan(t *testing.T, ended []sdktrace.ReadOnlySpan) sdktrace.ReadOnlySpan {
	t.Helper()

	index := slices.IndexFunc(ended, func(span sdktrace.ReadOnlySpan) bool {
		return span.Name() == rootSpanName
	})
	require.GreaterOrEqual(t, index, 0)

	return ended[index]
}

func TestRouterAttributionSpans(t *testing.T) {
	// Route attributes are wired when the router is built, so tracing goes on first.
	recorded := testutils.Telemetry(t)
	spans := recorded.Spans
	handler := attributionRouter(t)

	proxied := []string{
		"proxy.forward > proxy.copy_body",
		"proxy.forward > proxy.upstream",
	}

	tests := []struct {
		name    string
		method  string
		url     string
		handler events.HandlerKind
		route   string
		tree    []string
	}{
		{
			name:    "mock",
			method:  http.MethodGet,
			url:     "http://localhost/mock",
			handler: events.HandlerMock,
			route:   "mock",
			tree:    []string{"request > mock.serve"},
		},
		{
			name:    "script",
			method:  http.MethodGet,
			url:     "http://localhost/script",
			handler: events.HandlerScript,
			route:   "script",
			tree: []string{
				"request > script.serve",
				"script.serve > script.lua.compile",
				"script.serve > script.lua.execute",
			},
		},
		{
			name:    "static file",
			method:  http.MethodGet,
			url:     "http://localhost/static/app.js",
			handler: events.HandlerStatic,
			route:   "static",
			tree:    []string{"request > static.serve"},
		},
		{
			name:    "static fall-through",
			method:  http.MethodGet,
			url:     "http://localhost/static/missing.js",
			handler: events.HandlerProxy,
			route:   "static",
			tree:    append([]string{"request > proxy.forward", "request > static.serve"}, proxied...),
		},
		{
			name:    "preflight",
			method:  http.MethodOptions,
			url:     "http://localhost/api",
			handler: events.HandlerOptions,
			route:   "default",
			tree:    []string{"request > options.preflight"},
		},
		{
			name:    "proxied request",
			method:  http.MethodGet,
			url:     "http://localhost/api",
			handler: events.HandlerProxy,
			route:   "default",
			tree:    append([]string{"request > proxy.forward"}, proxied...),
		},
		{
			name:    "rewrite",
			method:  http.MethodGet,
			url:     "http://localhost/old",
			handler: events.HandlerProxy,
			route:   "rewrite",
			tree:    append([]string{"request > rewrite.apply", "rewrite.apply > proxy.forward"}, proxied...),
		},
		{
			name:    "cache miss",
			method:  http.MethodGet,
			url:     "http://localhost/cached/spans",
			handler: events.HandlerProxy,
			route:   "default",
			tree: append([]string{
				"request > cache.lookup",
				"request > cache.store",
				"request > proxy.forward",
			}, proxied...),
		},
		{
			name:    "cache hit",
			method:  http.MethodGet,
			url:     "http://localhost/cached/spans",
			handler: events.HandlerCache,
			route:   "default",
			tree:    []string{"request > cache.lookup"},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			ended := serveTraced(t, spans, handler, testCase.method, testCase.url)

			assert.ElementsMatch(t, testCase.tree, spanTree(ended))
			attrs := rootSpan(t, ended).Attributes()
			assert.Contains(t, attrs, attribute.String("uncors.handler", testCase.handler.String()))
			assert.Contains(t, attrs, attribute.String("uncors.route.kind", testCase.route))
			assert.Contains(t, attrs, attribute.String("uncors.mapping.from", "http://localhost"))
		})
	}

	result := attribute.Key("uncors.result")

	assert.Equal(t, int64(1), recorded.Sum(t, "uncors.cache.lookups", result.String("hit")))
	assert.Equal(t, int64(1), recorded.Sum(t, "uncors.cache.lookups", result.String("miss")))
	assert.Equal(t, int64(1), recorded.Sum(t, "uncors.cache.stores"))
	assert.Equal(t, int64(1), recorded.Sum(t, "uncors.static.requests", result.String("served")))
	assert.Equal(t, int64(1), recorded.Sum(t, "uncors.static.requests", result.String("fall_through")))
	assert.Equal(t, int64(1), recorded.Sum(t, "uncors.mock.responses", result.String("served")))
	assert.Equal(t, int64(1), recorded.Sum(t, "uncors.script.executions", result.String("ok")))
	assert.Equal(t, int64(1), recorded.Sum(t, "uncors.rewrites"))

	t.Run("unmapped host", func(t *testing.T) {
		ended := serveTraced(t, spans, handler, http.MethodGet, "http://unknown.local/api")

		require.Len(t, ended, 1)

		root := rootSpan(t, ended)
		require.Len(t, root.Events(), 2)
		assert.Equal(t, "uncors.host_not_mapped", root.Events()[0].Name)
		assert.Equal(t, "exception", root.Events()[1].Name)
	})
}
