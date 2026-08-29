package proxy

import (
	"context"
	"net/http"
	"net/http/httputil"
	"sync"
	"time"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/rewrite"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/go-http-utils/headers"
)

// flushInterval bounds how long a proxied byte may sit in the transport buffer.
// httputil.ReverseProxy flushes immediately regardless for responses that stream
// (text/event-stream, or an unknown content length).
const flushInterval = 100 * time.Millisecond

type replacersKey struct{}

// exchange carries what the ReverseProxy hooks need about the inbound request:
// they run after routing, and neither of them can report an error.
type exchange struct {
	target *urlreplacer.Replacer
	source *urlreplacer.Replacer
	origin string
}

type Handler struct {
	replacers urlreplacer.ReplacerFactory
	http      contracts.HTTPClient
	output    contracts.ErrorOutput

	proxy *httputil.ReverseProxy

	// rewriteReplacers memoises the replacers built for rewrite hosts. Building
	// them compiles two regexps, and the set of hosts comes from the config, so
	// it is small and bounded.
	rewriteReplacers sync.Map
}

func NewProxyHandler(options ...HandlerOption) *Handler {
	handler := helpers.ApplyOptions(&Handler{}, options)

	if handler.replacers == nil {
		panic("ProxyHandler: ReplacerFactory is not configured")
	}

	if handler.output == nil {
		panic("ProxyHandler: Output is not configured")
	}

	if handler.http == nil {
		panic("ProxyHandler: HTTP client is not configured")
	}

	handler.proxy = &httputil.ReverseProxy{
		Rewrite:        rewriteRequest,
		ModifyResponse: modifyResponse,
		ErrorHandler:   handler.handleError,
		Transport:      roundTripper{client: handler.http},
		FlushInterval:  flushInterval,
		ErrorLog:       infra.StdLogger(),
	}

	return handler
}

func (h *Handler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	infra.HandlerFunc(h.Serve).ServeHTTP(response, request)
}

// Serve is the error returning form of ServeHTTP. Only failures to resolve the
// mapping are reported this way; upstream transport failures are rendered by the
// proxy's error handler.
func (h *Handler) Serve(response http.ResponseWriter, request *http.Request) error {
	target, source, err := h.createReplacers(request)
	if err != nil {
		h.output.Errorf("Proxy handler error: %v", err)

		return err
	}

	ctx := context.WithValue(request.Context(), replacersKey{}, &exchange{
		target: target,
		source: source,
		origin: request.Header.Get(headers.Origin),
	})

	h.proxy.ServeHTTP(response, request.WithContext(ctx))

	return nil
}

func (h *Handler) handleError(writer http.ResponseWriter, request *http.Request, err error) {
	if request.Context().Err() != nil {
		// The client went away; there is nobody left to report to.
		return
	}

	h.output.Errorf("Proxy handler error: %v", err)
	http.Error(writer, "uncors: the request to the original source failed", http.StatusBadGateway)
}

func (h *Handler) createReplacers(req *http.Request) (*urlreplacer.Replacer, *urlreplacer.Replacer, error) {
	rewriteHost, err := rewrite.GetRewriteHost(req)
	if err != nil {
		return nil, nil, err
	}

	if rewriteHost == "" {
		return h.replacers.Make(req.URL)
	}

	return h.rewriteReplacersFor(req.URL.Host, rewriteHost)
}

func (h *Handler) rewriteReplacersFor(host, rewriteHost string) (*urlreplacer.Replacer, *urlreplacer.Replacer, error) {
	key := host + "->" + rewriteHost

	if cached, ok := h.rewriteReplacers.Load(key); ok {
		pair, _ := cached.(*replacerPair)

		return pair.target, pair.source, nil
	}

	target, err := urlreplacer.NewReplacer(host, rewriteHost)
	if err != nil {
		return nil, nil, err
	}

	source, err := urlreplacer.NewReplacer(rewriteHost, host)
	if err != nil {
		return nil, nil, err
	}

	h.rewriteReplacers.Store(key, &replacerPair{target: target, source: source})

	return target, source, nil
}

type replacerPair struct {
	target *urlreplacer.Replacer
	source *urlreplacer.Replacer
}

// roundTripper adapts the configured HTTP client to the transport the
// ReverseProxy drives.
type roundTripper struct {
	client contracts.HTTPClient
}

func (t roundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	// The proxy hands over a clone of the inbound request, which still carries
	// the server side RequestURI that http.Client refuses to send. The clone
	// belongs to this call, so clearing it in place is safe.
	request.RequestURI = ""

	return t.client.Do(request) //nolint:wrapcheck // the proxy's ErrorHandler reports it
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

func WithOutput(output contracts.ErrorOutput) HandlerOption {
	return func(m *Handler) {
		m.output = output
	}
}
