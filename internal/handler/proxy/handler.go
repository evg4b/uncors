package proxy

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"

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

func NewHandler(options ...HandlerOption) *Handler {
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

	// The upstream URL is built once, for the span and the logs.
	var target string

	if telemetry.Enabled() {
		target = helpers.LogURL(originalRequest.URL)
		recordTarget(req, originalRequest, target)
	}

	originalResponse, err := h.executeQuery(originalRequest, target)
	if err != nil {
		return err
	}

	if isProtocolSwitch(req, originalResponse) {
		return h.switchProtocols(originalResponse, resp, sourceReplacer, req)
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

	originalRequest.ContentLength = req.ContentLength

	copyHeaders(req.Header, originalRequest.Header, modificationsMap{
		headers.Origin:  replacer.ReplaceSoft,
		headers.Referer: replacer.ReplaceSoft,
	})

	removeHopByHopHeaders(originalRequest.Header)

	if hasToken(req.Header.Values(headers.TE), "trailers") {
		originalRequest.Header.Set(headers.TE, "trailers")
	}

	if infra.IsUpgrade(req.Header) {
		originalRequest.Header.Set(connectionHeader, "Upgrade")
		originalRequest.Header.Set(headers.Upgrade, req.Header.Get(headers.Upgrade))
	}

	return originalRequest, nil
}

func (h *Handler) makeUncorsResponse(
	original *http.Response,
	target http.ResponseWriter,
	replacer *urlreplacer.Replacer,
	req *http.Request,
) error {
	copyCookiesToSource(original, replacer, target)

	copyHeaders(original.Header, target.Header(), modificationsMap{
		headers.Location: replacer.ReplaceSoft,
	})

	removeHopByHopHeaders(target.Header())

	if hasNoBody(req, original) {
		if length := original.Header.Get(headers.ContentLength); length != "" {
			target.Header().Set(headers.ContentLength, length)
		}
	}

	switch {
	case tracking.OriginRejected(req.Context()):
		infra.WithholdCorsHeaders(target.Header())
	case strings.EqualFold(req.Method, http.MethodOptions):
		infra.WriteCorsHeadersForOptions(target.Header(), req.Header)
	default:
		infra.WriteCorsHeaders(target.Header(), req.Header.Get(headers.Origin))
	}

	target.WriteHeader(original.StatusCode)

	if !isStreaming(original) {
		return copyBody(req.Context(), target, original.Body)
	}

	controller := http.NewResponseController(target)
	_ = controller.Flush()

	return copyBody(req.Context(), flushWriter{writer: target, controller: controller}, original.Body)
}

func hasNoBody(request *http.Request, response *http.Response) bool {
	return request.Method == http.MethodHead || response.StatusCode == http.StatusNotModified
}

func isStreaming(response *http.Response) bool {
	if response.ContentLength == -1 {
		return true
	}

	mediaType, _, _ := strings.Cut(response.Header.Get(headers.ContentType), ";")

	return strings.EqualFold(strings.TrimSpace(mediaType), "text/event-stream")
}

type flushWriter struct {
	writer     io.Writer
	controller *http.ResponseController
}

func (w flushWriter) Write(data []byte) (int, error) {
	written, err := w.writer.Write(data)
	if err != nil {
		return written, err
	}

	_ = w.controller.Flush()

	return written, nil
}

const copyBufferSize = 32 * 1024

// copyBuffers holds the buffers bodies are copied through. Neither an
// upstream body nor the response writer offers a fast path, so io.Copy would
// allocate a buffer for every response.
var copyBuffers = sync.Pool{
	New: func() any {
		buffer := make([]byte, copyBufferSize)

		return &buffer
	},
}

func copyBody(ctx context.Context, target io.Writer, body io.Reader) error {
	_, span := telemetry.Start(ctx, "proxy.copy_body")
	defer span.End()

	buffer, _ := copyBuffers.Get().(*[]byte)
	defer copyBuffers.Put(buffer)

	written, err := io.CopyBuffer(target, body, *buffer)

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

// executeQuery sends the request upstream. The request carries the
// proxy.forward span, so the HTTP client's span nests under it.
func (h *Handler) executeQuery(request *http.Request, target string) (*http.Response, error) {
	originalResponse, err := h.http.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	if telemetry.Enabled() {
		recordUpstreamResponse(request.Context(), target, originalResponse)
	}

	return originalResponse, nil
}

func recordTarget(source, target *http.Request, targetURL string) {
	ctx := source.Context()

	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		attrs := []attribute.KeyValue{
			attribute.String("uncors.proxy.target", targetURL),
			attribute.String("uncors.proxy.target_host", target.URL.Host),
		}

		if rewriteHost, _ := rewrite.GetRewriteHost(source); rewriteHost != "" {
			attrs = append(attrs, attribute.String("uncors.proxy.rewrite_host", rewriteHost))
		}

		span.SetAttributes(attrs...)
	}

	telemetry.Debug(ctx, "forwarding request upstream",
		slog.String("method", target.Method), slog.String("target", targetURL))
}

// recordUpstreamResponse adds what the HTTP client's span leaves out, and
// warns about upstream server errors: the client sees them as uncors
// responses.
func recordUpstreamResponse(ctx context.Context, target string, response *http.Response) {
	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		span.SetAttributes(
			attribute.Int64("uncors.proxy.upstream_content_length", response.ContentLength),
			attribute.String("uncors.proxy.upstream_content_type", response.Header.Get(headers.ContentType)),
			semconv.NetworkProtocolVersion(telemetry.ProtocolVersion(response.ProtoMajor, response.ProtoMinor)),
		)
	}

	attrs := []slog.Attr{slog.String("target", target), slog.Int("status", response.StatusCode)}

	if response.StatusCode >= http.StatusInternalServerError {
		telemetry.Warn(ctx, "upstream responded with a server error", attrs...)

		return
	}

	telemetry.Debug(ctx, "upstream responded", attrs...)
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
