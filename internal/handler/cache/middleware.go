package cache

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/tracking"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/telemetry"
	"github.com/evg4b/uncors/pkg/urlt"
	"github.com/go-http-utils/headers"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

const ageHeader = "Age"

type Middleware struct {
	cache     contracts.Cache
	methods   []string
	pathGlobs config.CacheGlobs
}

func NewMiddleware(options ...MiddlewareOption) *Middleware {
	middleware := helpers.ApplyOptions(&Middleware{}, options)

	helpers.AssertIsDefined(middleware.cache, "Cache storage is not configured")

	return middleware
}

func (m *Middleware) ServeHTTP(writer contracts.ResponseWriter, request *contracts.Request, next contracts.Next) error {
	isCacheable, err := m.isCacheableRequest(request)
	if err != nil {
		return err
	}

	if !isCacheable {
		if telemetry.Enabled() {
			if span := trace.SpanFromContext(request.Context()); span.IsRecording() {
				span.AddEvent("uncors.cache.not_cacheable", trace.WithAttributes(
					attribute.StringSlice("uncors.cache.methods", m.methods),
					attribute.StringSlice("uncors.cache.globs", m.pathGlobs),
				))
			}
		}

		return next(writer, request)
	}

	handler := infra.HandlerFunc(func(w contracts.ResponseWriter, r *contracts.Request) error {
		return next(w, r)
	})

	return m.cacheRequest(writer, request, handler)
}

func (m *Middleware) cacheRequest(
	writer contracts.ResponseWriter,
	request *contracts.Request,
	next contracts.Handler,
) error {
	cacheKey := m.extractCacheKey(request)

	if m.serveCached(writer, request, cacheKey) {
		return nil
	}

	writer.EnableBodyCapture()

	err := next.ServeHTTP(writer, request)
	if err != nil || request.Context().Err() != nil {
		// The captured body may be truncated: an upstream reset or a client
		// disconnect can interrupt a response whose 2xx status is already sent.
		if telemetry.Enabled() {
			telemetry.Debug(request.Context(), "response not cached: the request did not complete",
				slog.String("key", cacheKey))
		}

		return err
	}

	m.storeResponse(request.Context(), request, cacheKey, writer.Captured())

	return nil
}

func (m *Middleware) serveCached(writer contracts.ResponseWriter, request *contracts.Request, baseKey string) bool {
	ctx, span := telemetry.Start(request.Context(), "cache.lookup")
	defer span.End()

	cachedResponse, cacheKey := m.getCachedResponse(request, baseKey)

	telemetry.CacheLookup(ctx, cachedResponse != nil)

	if telemetry.Enabled() {
		message := "cache miss"
		if cachedResponse != nil {
			message = "cache hit"
		}

		telemetry.Debug(ctx, message, slog.String("key", cacheKey))
	}

	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("uncors.cache.key", cacheKey),
			attribute.Bool("uncors.cache.hit", cachedResponse != nil),
		)
	}

	if cachedResponse == nil {
		return false
	}

	if span.IsRecording() {
		span.SetAttributes(
			semconv.HTTPResponseStatusCode(cachedResponse.Code),
			attribute.Int("uncors.cache.body_bytes", len(cachedResponse.Body)),
			attribute.Int("uncors.cache.headers", len(cachedResponse.Headers)),
		)
	}

	tracking.Claim(ctx, events.HandlerCache)

	err := m.writeCachedResponse(writer, request, cachedResponse)
	if err != nil {
		telemetry.Fail(span, err)
	}

	return true
}

func unstorableReason(capture contracts.ResponseCapture) (string, attribute.KeyValue) {
	switch {
	case !helpers.Is2xxCode(capture.StatusCode):
		return "the status is not 2xx", semconv.HTTPResponseStatusCode(capture.StatusCode)
	case capture.Truncated:
		return "the body is too large", attribute.Bool("uncors.cache.truncated", true)
	case len(capture.Header[headers.SetCookie]) > 0:
		return "the response sets cookies", attribute.Bool("uncors.cache.set_cookie", true)
	case forbidsStoring(capture.Header):
		return "the upstream forbids storing it",
			attribute.String("uncors.cache.cache_control", strings.Join(capture.Header[headers.CacheControl], ", "))
	case variesOnAll(capture.Header):
		return "the response varies on every request header", attribute.String("uncors.cache.vary", anyValue)
	}

	return "", attribute.KeyValue{}
}

func isStorable(ctx context.Context, key string, capture contracts.ResponseCapture) bool {
	reason, attr := unstorableReason(capture)
	if reason == "" {
		return true
	}

	if telemetry.Enabled() {
		if span := trace.SpanFromContext(ctx); span.IsRecording() {
			span.AddEvent("uncors.cache.not_stored", trace.WithAttributes(attr))
		}

		telemetry.Debug(ctx, "response not cached: "+reason, slog.String("key", key),
			slog.Int("status", capture.StatusCode), slog.Int("bytes", len(capture.Body)))
	}

	return false
}

func (m *Middleware) storeResponse(
	ctx context.Context,
	request *contracts.Request,
	baseKey string,
	capture contracts.ResponseCapture,
) {
	if !isStorable(ctx, baseKey, capture) {
		return
	}

	vary := varyNames(capture.Header)
	key := variantKey(baseKey, vary, request.Header)

	_, span := telemetry.Start(ctx, "cache.store")
	defer span.End()

	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("uncors.cache.key", key),
			semconv.HTTPResponseStatusCode(capture.StatusCode),
			attribute.Int("uncors.cache.body_bytes", len(capture.Body)),
		)
	}

	response := contracts.CachedResponse{
		Code:    capture.StatusCode,
		Body:    capture.Body,
		Headers: storedHeaders(capture.Header),
		Stored:  time.Now(),
	}

	stored := m.cache.Set(key, response)
	if stored && len(vary) > 0 {
		stored = m.cache.Set(baseKey, contracts.CachedResponse{Vary: vary})
	}

	if !stored {
		telemetry.CacheRejected(ctx)

		if span.IsRecording() {
			span.SetAttributes(attribute.Bool("uncors.cache.rejected", true))
			span.AddEvent("uncors.cache.rejected", trace.WithAttributes(
				attribute.Int64("uncors.cache.cost", CalcCost(&response)),
			))
		}

		if telemetry.Enabled() {
			telemetry.Debug(ctx, "response not cached: the cache storage rejected it",
				slog.String("key", key), slog.Int("bytes", len(capture.Body)))
		}

		return
	}

	telemetry.CacheStored(ctx, len(capture.Body))

	if telemetry.Enabled() {
		telemetry.Debug(ctx, "response cached",
			slog.String("key", key), slog.Int("status", capture.StatusCode), slog.Int("bytes", len(capture.Body)))
	}
}

func storedHeaders(header http.Header) []contracts.CachedHeader {
	cached := make([]contracts.CachedHeader, 0, len(header))

	for name, values := range header {
		// It names the trace of the request that filled the cache.
		if name == telemetry.TraceIDHeader || name == headers.SetCookie || isCorsHeader(name) {
			continue
		}

		cached = append(cached, contracts.CachedHeader{Name: name, Value: values})
	}

	sort.Slice(cached, func(i, j int) bool {
		return cached[i].Name < cached[j].Name
	})

	return cached
}

func (m *Middleware) writeCachedResponse(
	writer contracts.ResponseWriter,
	request *contracts.Request,
	cachedResponse *contracts.CachedResponse,
) error {
	header := writer.Header()

	for _, cachedHeader := range cachedResponse.Headers {
		for _, value := range cachedHeader.Value {
			header.Add(cachedHeader.Name, value)
		}
	}

	upstreamAge, _ := strconv.ParseInt(header.Get(ageHeader), 10, 64)
	header.Set(ageHeader, strconv.FormatInt(upstreamAge+int64(time.Since(cachedResponse.Stored)/time.Second), 10))

	switch {
	case tracking.OriginRejected(request.Context()):
		infra.WithholdCorsHeaders(header)
	case request.Method == http.MethodOptions:
		infra.WriteCorsHeadersForOptions(header, request.Header)
	default:
		infra.WriteCorsHeaders(header, request.Header.Get(headers.Origin))
	}

	writer.WriteHeader(cachedResponse.Code)

	if len(cachedResponse.Body) == 0 {
		return nil
	}

	_, err := writer.Write(cachedResponse.Body)

	return err
}

func (m *Middleware) isCacheableRequest(request *contracts.Request) (bool, error) {
	if !slices.Contains(m.methods, request.Method) {
		return false, nil
	}

	for _, pattern := range m.pathGlobs {
		ok, err := doublestar.PathMatch(pattern, request.URL.Path)
		if err != nil {
			return false, err
		}

		if ok {
			return true, nil
		}
	}

	return false, nil
}

func (m *Middleware) extractCacheKey(request *contracts.Request) string {
	requestURL := request.URL
	values := urlt.URL_Query(requestURL)

	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}

	sort.Strings(names)

	items := make([]string, 0, len(values))
	for _, name := range names {
		for _, value := range values[name] {
			items = append(items, url.QueryEscape(name)+"="+url.QueryEscape(value))
		}
	}

	// The store is shared by every mapping of a configuration generation, so the
	// key must identify the listener (scheme and port), not just the host name.
	return fmt.Sprintf("[%s]%s://%s%s?%s#%s",
		request.Method, requestURL.Scheme, strings.ToLower(requestURL.Host), requestURL.Path,
		strings.Join(items, "&"), acceptEncoding(request.Header))
}

func (m *Middleware) getCachedResponse(request *contracts.Request, baseKey string) (*contracts.CachedResponse, string) {
	cachedResponse, ok := m.cache.Get(baseKey)
	if !ok {
		return nil, baseKey
	}

	if len(cachedResponse.Vary) == 0 {
		return &cachedResponse, baseKey
	}

	key := variantKey(baseKey, cachedResponse.Vary, request.Header)

	variant, ok := m.cache.Get(key)
	if !ok || len(variant.Vary) > 0 {
		return nil, key
	}

	return &variant, key
}

type MiddlewareOption = func(*Middleware)

func WithMethods(methods []string) MiddlewareOption {
	return func(m *Middleware) {
		m.methods = methods
	}
}

func WithGlobs(globs config.CacheGlobs) MiddlewareOption {
	return func(m *Middleware) {
		m.pathGlobs = globs
	}
}

func WithCacheStorage(cache contracts.Cache) MiddlewareOption {
	return func(m *Middleware) {
		m.cache = cache
	}
}
