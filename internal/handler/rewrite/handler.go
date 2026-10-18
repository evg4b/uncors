package rewrite

import (
	"context"
	"log/slog"
	"maps"
	"net/url"
	"slices"
	"strings"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/telemetry"
	"github.com/evg4b/uncors/internal/tracking"
	"github.com/evg4b/uncors/pkg/urlt"
	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Middleware struct {
	rewrite *config.RewritingOption
}

func NewMiddleware(options ...MiddlewareOption) *Middleware {
	middleware := helpers.ApplyOptions(&Middleware{}, options)

	helpers.AssertIsDefined(middleware.rewrite, "RewriteMiddleware: Rewriting options are not configured")

	return middleware
}

func (m *Middleware) ServeHTTP(writer contracts.ResponseWriter, request *contracts.Request, next contracts.Next) error {
	request, span := telemetry.StartRequest(request, "rewrite.apply")
	defer span.End()

	from := request.URL.Path

	m.rewriteURL(request)

	to := request.URL.Path
	if m.rewrite.Host != (urlt.Host{}) {
		to = m.rewrite.Host.HostPort() + to
	}

	tracking.Rewrote(request.Context(), from, to)
	telemetry.Rewritten(request.Context())

	if span.IsRecording() {
		m.annotate(span, request, from, to)
	}

	if telemetry.Enabled() {
		telemetry.Debug(request.Context(), "rewrote request",
			slog.String("pattern", m.rewrite.From), slog.String("from", from), slog.String("to", to))
	}

	return next(writer, m.rewriteRequest(request))
}

func (m *Middleware) annotate(span trace.Span, request *contracts.Request, from, to string) {
	attrs := []attribute.KeyValue{
		attribute.String("uncors.rewrite.pattern", m.rewrite.From),
		attribute.String("uncors.rewrite.template", m.rewrite.To),
		attribute.String("uncors.rewrite.from", from),
		attribute.String("uncors.rewrite.to", to),
	}

	if m.rewrite.Host != (urlt.Host{}) {
		attrs = append(attrs, attribute.String("uncors.rewrite.host", m.rewrite.Host.HostPort()))
	}

	if vars := mux.Vars(request); len(vars) > 0 {
		pairs := make([]string, 0, len(vars))
		for _, key := range slices.Sorted(maps.Keys(vars)) {
			pairs = append(pairs, key+"="+vars[key])
		}

		attrs = append(attrs, attribute.StringSlice("uncors.rewrite.vars", pairs))
	}

	span.SetAttributes(attrs...)
}

func (m *Middleware) rewriteURL(request *contracts.Request) {
	// A reference with a path resolves to the reference's own query, so the
	// original query has to be carried over explicitly.
	clonedURL := &url.URL{
		Path:       replace(m.rewrite.To, mux.Vars(request)),
		RawQuery:   request.URL.RawQuery,
		ForceQuery: request.URL.ForceQuery,
	}
	request.URL = urlt.URL_ResolveReference(request.URL, clonedURL)
}

func (m *Middleware) rewriteRequest(request *contracts.Request) *contracts.Request {
	if m.rewrite.Host == (urlt.Host{}) {
		return request
	}

	return request.WithContext(
		context.WithValue(request.Context(), RewriteHostKey, m.rewrite.Host.HostPort()),
	)
}

func replace(template string, data map[string]string) string {
	var builder strings.Builder

	builder.Grow(len(template))

	for {
		start := strings.IndexByte(template, '{')
		if start < 0 {
			break
		}

		length := strings.IndexByte(template[start+1:], '}')
		if length < 0 {
			break
		}

		end := start + 1 + length

		value, ok := data[template[start+1:end]]
		if !ok {
			builder.WriteString(template[:start+1])
			template = template[start+1:]

			continue
		}

		builder.WriteString(template[:start])
		builder.WriteString(value)
		template = template[end+1:]
	}

	builder.WriteString(template)

	return builder.String()
}

type MiddlewareOption = func(*Middleware)

func WithRewritingOptions(rewrite *config.RewritingOption) MiddlewareOption {
	return func(h *Middleware) {
		h.rewrite = rewrite
	}
}
