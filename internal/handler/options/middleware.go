package options

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/tracking"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

type Middleware struct {
	headers map[string]string
	code    int
}

func NewMiddleware(options ...MiddlewareOption) *Middleware {
	return helpers.ApplyOptions(&Middleware{}, options)
}

func (m *Middleware) ServeHTTP(resp contracts.ResponseWriter, req *contracts.Request, next contracts.Next) error {
	if strings.EqualFold(req.Method, http.MethodOptions) {
		req, span := telemetry.StartRequest(req, "options.preflight")
		defer span.End()

		tracking.Claim(req.Context(), events.HandlerOptions)

		if span.IsRecording() {
			span.SetAttributes(
				semconv.HTTPResponseStatusCode(helpers.NormaliseStatusCode(m.code)),
				attribute.Int("uncors.options.custom_headers", len(m.headers)),
			)
		}

		m.handle(resp, req)

		if telemetry.Enabled() {
			telemetry.Debug(req.Context(), "answered preflight request",
				slog.String("origin", req.Header.Get("Origin")),
				slog.String("requested_method", req.Header.Get("Access-Control-Request-Method")),
				slog.Int("status", helpers.NormaliseStatusCode(m.code)),
			)
		}

		return nil
	}

	return next(resp, req)
}

func (m *Middleware) handle(resp http.ResponseWriter, req *http.Request) {
	infra.WriteCorsHeadersForOptions(resp.Header(), req.Header)

	for key, value := range m.headers {
		resp.Header().Set(key, value)
	}

	resp.WriteHeader(helpers.NormaliseStatusCode(m.code))
}

type MiddlewareOption = func(*Middleware)

func WithHeaders(headers map[string]string) MiddlewareOption {
	return func(m *Middleware) {
		m.headers = headers
	}
}

func WithCode(code int) MiddlewareOption {
	return func(m *Middleware) {
		m.code = code
	}
}
