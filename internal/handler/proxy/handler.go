package proxy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/telemetry"
	"github.com/evg4b/uncors/internal/tracking"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/rewrite"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/evg4b/uncors/pkg/urlt"
	"github.com/go-http-utils/headers"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

type Handler struct {
	replacers urlreplacer.ReplacerFactory
	http      contracts.HTTPClient
}

func NewProxyHandler(options ...HandlerOption) *Handler {
	middleware := helpers.ApplyOptions(&Handler{}, options)

	helpers.AssertIsDefined(middleware.replacers, "ProxyHandler: ReplacerFactory is not configured")
	helpers.AssertIsDefined(middleware.http, "ProxyHandler: Http client is not configured")

	return middleware
}

func (h *Handler) ServeHTTP(response contracts.ResponseWriter, request *contracts.Request) error {
	request, span := telemetry.StartRequest(request, "proxy.forward")
	defer span.End()

	tracking.Claim(request.Context(), events.HandlerProxy)

	err := h.handle(response, request)
	if err != nil {
		if request.Context().Err() == nil {
			telemetry.Fail(span, err)
		}

		return err
	}

	return nil
}

func (h *Handler) handle(resp http.ResponseWriter, req *http.Request) error {
	targetReplacer, sourceReplacer, err := h.createReplacers(req)
	if err != nil {
		return err
	}

	originalRequest, err := h.makeOriginalRequest(req, targetReplacer)
	if err != nil {
		return fmt.Errorf("failed to create request to original source: %w", err)
	}

	if span := trace.SpanFromContext(req.Context()); span.IsRecording() {
		annotateTarget(span, req, originalRequest)
	}

	originalResponse, err := h.executeQuery(req.Context(), originalRequest)
	if err != nil {
		return err
	}

	defer helpers.CloseSafe(originalResponse.Body)

	err = h.makeUncorsResponse(originalResponse, resp, sourceReplacer, req)
	if err != nil {
		return fmt.Errorf("failed to make uncors response: %w", err)
	}

	return nil
}

func (h *Handler) makeOriginalRequest(
	req *http.Request,
	replacer *urlreplacer.Replacer,
) (*http.Request, error) {
	url, err := replacer.Replace(urlt.URL_String(req.URL))
	if err != nil {
		return nil, fmt.Errorf("failed to replace URL: %w", err)
	}

	//nolint:gosec // G704: forwarding to user-configured target is intentional
	originalRequest, err := http.NewRequestWithContext(req.Context(), req.Method, url, req.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to make request to original server: %w", err)
	}

	err = copyHeaders(req.Header, originalRequest.Header, modificationsMap{
		headers.Origin:  replacer.Replace,
		headers.Referer: replacer.Replace,
	})
	if err != nil {
		return nil, err
	}

	copyCookiesToTarget(req, replacer, originalRequest)

	return originalRequest, nil
}

func (h *Handler) makeUncorsResponse(
	original *http.Response,
	target http.ResponseWriter,
	replacer *urlreplacer.Replacer,
	req *http.Request,
) error {
	copyCookiesToSource(original, replacer, target)

	err := copyHeaders(original.Header, target.Header(), modificationsMap{
		headers.Location: func(s string) (string, error) { //nolint:unparam
			return replacer.ReplaceSoft(s), nil
		},
	})
	if err != nil {
		return err
	}

	origin := req.Header.Get(headers.Origin)
	infra.WriteCorsHeaders(target.Header(), origin)

	target.WriteHeader(original.StatusCode)

	return copyBody(req.Context(), target, original.Body)
}

func copyBody(ctx context.Context, target io.Writer, body io.Reader) error {
	_, span := telemetry.Start(ctx, "proxy.copy_body")
	defer span.End()

	written, err := io.Copy(target, body)

	if span.IsRecording() {
		span.SetAttributes(attribute.Int64("uncors.proxy.body_bytes", written))
	}

	if err != nil {
		err = fmt.Errorf("failed to copy body to response: %w", err)
		telemetry.Fail(span, err)

		return err
	}

	return nil
}

func (h *Handler) createReplacers(req *http.Request) (*urlreplacer.Replacer, *urlreplacer.Replacer, error) {
	rewriteHost, err := rewrite.GetRewriteHost(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get rewrite host: %w", err)
	}

	if rewriteHost == "" {
		targetReplacer, sourceReplacer, err := h.replacers.Make(req.URL)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to transform general url: %w", err)
		}

		return targetReplacer, sourceReplacer, nil
	}

	target, err := urlreplacer.NewReplacer(req.URL.Host, rewriteHost)
	if err != nil {
		return nil, nil, err
	}

	source, err := urlreplacer.NewReplacer(rewriteHost, req.URL.Host)
	if err != nil {
		return nil, nil, err
	}

	return target, source, nil
}

func (h *Handler) executeQuery(ctx context.Context, request *http.Request) (*http.Response, error) {
	ctx, span := telemetry.Start(ctx, "proxy.upstream")
	defer span.End()

	if span.IsRecording() {
		span.SetAttributes(
			semconv.HTTPRequestMethodKey.String(request.Method),
			semconv.URLFull(helpers.LogURL(request.URL)),
		)
		// The HTTP client span, with its DNS, connection, and TLS spans, nests
		// under this one.
		request = request.WithContext(ctx)
	}

	originalResponse, err := h.http.Do(request)
	if err != nil {
		telemetry.UpstreamRequest(ctx, request.Method, request.URL.Host, 0, err)

		err = fmt.Errorf("failed to execute request: %w", err)
		telemetry.Fail(span, err)

		return nil, err
	}

	telemetry.UpstreamRequest(ctx, request.Method, request.URL.Host, originalResponse.StatusCode, nil)

	if span.IsRecording() {
		span.SetAttributes(
			semconv.HTTPResponseStatusCode(originalResponse.StatusCode),
			attribute.Int64("uncors.proxy.upstream_content_length", originalResponse.ContentLength),
			attribute.String("uncors.proxy.upstream_content_type", originalResponse.Header.Get(headers.ContentType)),
			semconv.NetworkProtocolVersion(strconv.Itoa(originalResponse.ProtoMajor)+"."+
				strconv.Itoa(originalResponse.ProtoMinor)),
		)
	}

	return originalResponse, nil
}

func annotateTarget(span trace.Span, source, target *http.Request) {
	attrs := []attribute.KeyValue{
		attribute.String("uncors.proxy.target", helpers.LogURL(target.URL)),
		attribute.String("uncors.proxy.target_host", target.URL.Host),
	}

	if rewriteHost, _ := rewrite.GetRewriteHost(source); rewriteHost != "" {
		attrs = append(attrs, attribute.String("uncors.proxy.rewrite_host", rewriteHost))
	}

	span.SetAttributes(attrs...)
}

type HandlerOption = func(*Handler)

func WithURLReplacerFactory(replacerFactory urlreplacer.ReplacerFactory) HandlerOption {
	return func(m *Handler) {
		m.replacers = replacerFactory
	}
}

func WithHTTPClient(http contracts.HTTPClient) HandlerOption {
	return func(m *Handler) {
		m.http = http
	}
}
