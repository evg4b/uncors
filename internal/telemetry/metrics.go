package telemetry

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

// Values of the uncors.result metric attribute.
const (
	ResultOK          = "ok"
	ResultError       = "error"
	ResultHit         = "hit"
	ResultMiss        = "miss"
	ResultServed      = "served"
	ResultFallThrough = "fall_through"
	ResultCancelled   = "cancelled"
	ResultRecorded    = "recorded"
	ResultDropped     = "dropped"
)

const (
	handlerKey = attribute.Key("uncors.handler")
	resultKey  = attribute.Key("uncors.result")
)

// RequestMetrics describes a request the server finished.
type RequestMetrics struct {
	Method        string
	Handler       string
	Status        int
	Cancelled     bool
	Duration      time.Duration
	ResponseBytes int64
}

type instruments struct {
	requests         metric.Int64Counter
	activeRequests   metric.Int64UpDownCounter
	requestDuration  metric.Float64Histogram
	responseSize     metric.Int64Histogram
	upstreamRequests metric.Int64Counter
	cacheLookups     metric.Int64Counter
	cacheStores      metric.Int64Counter
	cacheStoredBytes metric.Int64Counter
	mockResponses    metric.Int64Counter
	mockDelay        metric.Float64Histogram
	scriptRuns       metric.Int64Counter
	scriptDuration   metric.Float64Histogram
	staticRequests   metric.Int64Counter
	rewrites         metric.Int64Counter
	harEntries       metric.Int64Counter
	harFlushes       metric.Int64Counter
	configReloads    metric.Int64Counter
	tlsCertificates  metric.Int64Counter
}

// Instrument errors are ignored: the meter returns a working instrument even
// when it reports one, and the names here are fixed.
func newInstruments(meter metric.Meter) *instruments {
	counter := func(name, unit, description string) metric.Int64Counter {
		instrument, _ := meter.Int64Counter(name, metric.WithUnit(unit), metric.WithDescription(description))

		return instrument
	}

	seconds := func(name, description string) metric.Float64Histogram {
		instrument, _ := meter.Float64Histogram(name, metric.WithUnit("s"), metric.WithDescription(description))

		return instrument
	}

	activeRequests, _ := meter.Int64UpDownCounter("http.server.active_requests",
		metric.WithUnit("{request}"), metric.WithDescription("Requests being served."))
	responseSize, _ := meter.Int64Histogram("http.server.response.body.size",
		metric.WithUnit("By"), metric.WithDescription("Size of the response bodies sent to clients."))

	return &instruments{
		requests:         counter("uncors.requests", "{request}", "Requests received, by the handler that answered."),
		activeRequests:   activeRequests,
		requestDuration:  seconds("http.server.request.duration", "Time to serve a request."),
		responseSize:     responseSize,
		upstreamRequests: counter("uncors.upstream.requests", "{request}", "Requests sent to upstream servers."),
		cacheLookups:     counter("uncors.cache.lookups", "{lookup}", "Cache lookups, by hit or miss."),
		cacheStores:      counter("uncors.cache.stores", "{response}", "Responses stored in the cache."),
		cacheStoredBytes: counter("uncors.cache.stored_bytes", "By", "Bytes of response bodies stored in the cache."),
		mockResponses:    counter("uncors.mock.responses", "{response}", "Mock responses, served or cancelled."),
		mockDelay:        seconds("uncors.mock.delay", "Configured delays of the mock responses served."),
		scriptRuns:       counter("uncors.script.executions", "{execution}", "Lua script executions."),
		scriptDuration:   seconds("uncors.script.duration", "Time to compile and run a Lua script."),
		staticRequests:   counter("uncors.static.requests", "{request}", "Static requests, served or passed on."),
		rewrites:         counter("uncors.rewrites", "{request}", "Requests rewritten."),
		harEntries:       counter("uncors.har.entries", "{entry}", "HAR entries, recorded or dropped."),
		harFlushes:       counter("uncors.har.flushes", "{flush}", "HAR file writes."),
		configReloads:    counter("uncors.config.reloads", "{reload}", "Configuration reloads."),
		tlsCertificates:  counter("uncors.tls.certificates", "{certificate}", "Host certificates generated."),
	}
}

// The recording functions below return at once when tracing is off, so
// they can be called on the request path without a guard.

func RequestStarted(ctx context.Context) {
	if state := current.Load(); state != nil {
		state.metrics.activeRequests.Add(ctx, 1)
	}
}

func RequestFinished(ctx context.Context, request RequestMetrics) {
	state := current.Load()
	if state == nil {
		return
	}

	attrs := metric.WithAttributes(
		semconv.HTTPRequestMethodKey.String(request.Method),
		semconv.HTTPResponseStatusCode(request.Status),
		handlerKey.String(request.Handler),
		attribute.Bool("uncors.cancelled", request.Cancelled),
	)

	state.metrics.activeRequests.Add(ctx, -1)
	state.metrics.requests.Add(ctx, 1, attrs)
	state.metrics.requestDuration.Record(ctx, request.Duration.Seconds(), attrs)
	state.metrics.responseSize.Record(ctx, request.ResponseBytes, attrs)
}

func UpstreamRequest(ctx context.Context, method, host string, status int, err error) {
	state := current.Load()
	if state == nil {
		return
	}

	attrs := []attribute.KeyValue{
		semconv.HTTPRequestMethodKey.String(method),
		semconv.ServerAddress(host),
	}

	if err != nil {
		attrs = append(attrs, semconv.ErrorTypeKey.String(errorType(err)))
	} else {
		attrs = append(attrs, semconv.HTTPResponseStatusCode(status))
	}

	state.metrics.upstreamRequests.Add(ctx, 1, metric.WithAttributes(attrs...))
}

func CacheLookup(ctx context.Context, hit bool) {
	result := ResultMiss
	if hit {
		result = ResultHit
	}

	count(ctx, func(m *instruments) metric.Int64Counter { return m.cacheLookups }, result)
}

func CacheStored(ctx context.Context, bytes int) {
	if state := current.Load(); state != nil {
		state.metrics.cacheStores.Add(ctx, 1)
		state.metrics.cacheStoredBytes.Add(ctx, int64(bytes))
	}
}

func MockServed(ctx context.Context, status int, delay time.Duration, cancelled bool) {
	state := current.Load()
	if state == nil {
		return
	}

	result := ResultServed
	if cancelled {
		result = ResultCancelled
	}

	state.metrics.mockResponses.Add(ctx, 1, metric.WithAttributes(
		resultKey.String(result),
		semconv.HTTPResponseStatusCode(status),
	))

	if delay > 0 && !cancelled {
		state.metrics.mockDelay.Record(ctx, delay.Seconds())
	}
}

func ScriptExecuted(ctx context.Context, duration time.Duration, err error) {
	state := current.Load()
	if state == nil {
		return
	}

	attrs := metric.WithAttributes(resultKey.String(resultOf(err)))

	state.metrics.scriptRuns.Add(ctx, 1, attrs)
	state.metrics.scriptDuration.Record(ctx, duration.Seconds(), attrs)
}

// StaticRequest takes ResultServed, ResultFallThrough, or ResultError.
func StaticRequest(ctx context.Context, result string) {
	count(ctx, func(m *instruments) metric.Int64Counter { return m.staticRequests }, result)
}

func Rewritten(ctx context.Context) {
	if state := current.Load(); state != nil {
		state.metrics.rewrites.Add(ctx, 1)
	}
}

func HAREntry(ctx context.Context, recorded bool) {
	result := ResultDropped
	if recorded {
		result = ResultRecorded
	}

	count(ctx, func(m *instruments) metric.Int64Counter { return m.harEntries }, result)
}

func HARFlushed(ctx context.Context, err error) {
	count(ctx, func(m *instruments) metric.Int64Counter { return m.harFlushes }, resultOf(err))
}

func ConfigReloaded(ctx context.Context, err error) {
	count(ctx, func(m *instruments) metric.Int64Counter { return m.configReloads }, resultOf(err))
}

func TLSCertificate(ctx context.Context, err error) {
	count(ctx, func(m *instruments) metric.Int64Counter { return m.tlsCertificates }, resultOf(err))
}

func count(ctx context.Context, instrument func(*instruments) metric.Int64Counter, result string) {
	if state := current.Load(); state != nil {
		instrument(state.metrics).Add(ctx, 1, metric.WithAttributes(resultKey.String(result)))
	}
}

func resultOf(err error) string {
	if err != nil {
		return ResultError
	}

	return ResultOK
}

func errorType(err error) string {
	switch {
	case errors.Is(err, context.Canceled):
		return "canceled"
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	default:
		for next := errors.Unwrap(err); next != nil; next = errors.Unwrap(next) {
			err = next
		}

		return fmt.Sprintf("%T", err)
	}
}
