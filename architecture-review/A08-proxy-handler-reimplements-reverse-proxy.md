# A08 — The proxy handler hand-rolls a reverse proxy instead of using `httputil.ReverseProxy`

**Severity:** High
**Area:** HTTP / proxy architecture

---

## 1. What is wrong with the current approach

`internal/handler/proxy` implements request forwarding from scratch on top of
`http.Client`:

```go
// internal/handler/proxy/handler.go:47-73
targetReplacer, sourceReplacer, err := h.createReplacers(req)
originalRequest, err := h.makeOriginalRequest(req, targetReplacer)
originalResponse, err := h.executeQuery(originalRequest)   // h.http.Do(request)
defer helpers.CloseSafe(originalResponse.Body)
err = h.makeUncorsResponse(originalResponse, resp, sourceReplacer, req)
```

What this re-implements, and what it gets wrong or omits compared to
`net/http/httputil.ReverseProxy`:

| Concern | `ReverseProxy` | This implementation |
| --- | --- | --- |
| Streaming / periodic flush | `FlushInterval`, flushes on `Flusher` | none — `io.Copy` into a non-flushing writer ([A07](A07-response-writer-drops-streaming-capabilities.md)) |
| `Connection: Upgrade` (WebSocket) | supported via hijack | not possible |
| Hop-by-hop header stripping | RFC-compliant (`Connection`, `Keep-Alive`, `TE`, `Trailer`, `Upgrade`, `Proxy-*`) | only `Cookie`, `Set-Cookie`, `Content-Length` are skipped ([`internal/handler/proxy/helpers.go:15`](../internal/handler/proxy/helpers.go#L15)) |
| `X-Forwarded-For` / `Forwarded` | `ProxyRequest.SetXForwarded()` | absent |
| Trailers | forwarded | dropped |
| 1xx informational responses | forwarded via `httptrace` | dropped |
| Error handling | `ErrorHandler` hook, 502 | 500 with a stack trace ([A14](A14-http-error-page-leaks-stack-traces.md)) |
| Client disconnect | context cancellation propagated | partially — checked only in the error branch ([`internal/handler/proxy/handler.go:34`](../internal/handler/proxy/handler.go#L34)) |
| Response body close | `defer resp.Body.Close()` | `helpers.CloseSafe` which **panics** on close error ([`internal/helpers/closer.go:6`](../internal/helpers/closer.go#L6)) |

Two further specifics worth calling out:

**Hop-by-hop headers are forwarded verbatim.** `copyHeaders` copies every header
except three. That means a client's `Connection: keep-alive`,
`Transfer-Encoding`, `Upgrade`, or `Proxy-Authorization` is passed straight to the
upstream, and the upstream's hop-by-hop headers are passed back to the client.
This is a spec violation that produces real, hard-to-diagnose bugs (double
`Transfer-Encoding`, upstreams that see an `Upgrade` they cannot honour).

**Cookies are mutated in place on the shared request.**

```go
// internal/handler/proxy/helpers.go:41-48
for _, cookie := range target.Cookies() {
	cookie.Secure = replacer.IsTargetSecure()
	cookie.Domain = replacer.ReplaceSoft(cookie.Domain)
	http.SetCookie(source, cookie)
}
```

`(*http.Response).Cookies()` and `(*http.Request).Cookies()` return freshly
parsed values, so this is currently safe — but it reads as in-place mutation and
would become a data race the moment those values were cached or shared.

**The URL rewriting is string-based.** `makeOriginalRequest` renders the incoming
`*url.URL` to a string, runs a regexp replacement over it, and re-parses it
([`internal/handler/proxy/handler.go:79`](../internal/handler/proxy/handler.go#L79)).
Since only the *host* can contain placeholders
([`internal/urlreplacer/helpers.go:89`](../internal/urlreplacer/helpers.go#L89) strips
everything after the authority), rewriting the whole URL string means every
proxied request pays a regexp match plus repeated `strings.ReplaceAll` over the
pattern ([`internal/urlreplacer/replacer.go:85`](../internal/urlreplacer/replacer.go#L85)),
and any escaping subtlety in the path or query has to be handled by the
`pkg/urlt` fork ([A17](A17-pkg-urlt-is-a-fork-of-net-url.md)) instead of by
`net/url`.

## 2. Why it is an architectural problem

- **The core value proposition is proxying**, and it is the one component built
  without the standard library's implementation. Every gap above is a class of
  user-visible bug (streaming, websockets, header correctness) that the stdlib
  has already solved and keeps solving as the HTTP spec evolves.
- **The transformation logic is entangled with the transport logic.** CORS
  rewriting, cookie domain rewriting and URL replacement are policy; connection
  management, buffering and error handling are mechanism. Today they live in one
  120-line handler, so improving the mechanism means rewriting the policy.
- **It cannot be extended.** Adding HTTP/2 to upstreams, connection reuse tuning,
  retry-on-idempotent, or upgrade support all require restructuring rather than
  configuration.

## 3. What the recommended approach is instead

Rebuild the proxy handler around `httputil.ReverseProxy`, keeping all uncors
policy in the two documented hooks:

```go
rp := &httputil.ReverseProxy{
	Rewrite: func(pr *httputil.ProxyRequest) {
		target, source := replacers.For(pr.In)          // existing urlreplacer
		pr.Out.URL = target.ApplyTo(pr.In.URL)          // structured, not string-based
		pr.Out.Host = pr.Out.URL.Host
		pr.SetXForwarded()
		rewriteOriginAndReferer(pr.Out.Header, target)  // existing modificationsMap logic
		rewriteRequestCookies(pr.Out, target)
	},
	ModifyResponse: func(res *http.Response) error {
		rewriteLocation(res.Header, source)
		rewriteResponseCookies(res, source)
		infra.WriteCorsHeaders(res.Header, originOf(res.Request))
		return nil
	},
	ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
		output.Errorf("proxy error: %v", err)
		http.Error(w, "upstream unreachable", http.StatusBadGateway)
	},
	Transport:     sharedTransport,        // see A09
	FlushInterval: 100 * time.Millisecond, // -1 for immediate flush on SSE
	ErrorLog:      infra.StdLogger(),
}
```

`ReverseProxy.Rewrite` (Go 1.20+) is specifically the successor to `Director` and
gives correct hop-by-hop handling, `X-Forwarded-*`, trailers, 1xx forwarding and
upgrade support for free.

Additionally:

- **Make `urlreplacer` operate on `*url.URL`, not strings.** Since placeholders
  are host-only, the replacer only needs to transform `URL.Host` (and pick the
  scheme). `Path`, `RawQuery` and `Fragment` should be copied across untouched.
  This removes the render→regexp→parse round trip from the hot path and removes
  most of the reason `pkg/urlt` exists.
- **Replace `helpers.CloseSafe` in this path** with a plain deferred `Close()`;
  panicking because an upstream body failed to close is not a reasonable
  response.
- **Set `FlushInterval: -1` when the upstream response is `text/event-stream`**
  (ReverseProxy does this automatically).

## 4. Why the proposed approach is better

- **Correctness by default** for the long tail of HTTP semantics that a hand-rolled
  forwarder inevitably misses, maintained by the Go team rather than by this
  project.
- **Streaming, SSE and WebSocket** become supported features rather than known
  gaps.
- **Policy is isolated in two small functions**, which makes the CORS/cookie/URL
  rewriting logic far easier to test — `Rewrite` and `ModifyResponse` are pure
  functions over a request/response.
- **Less code**: `handler.go` + `helpers.go` (≈240 lines) collapse to roughly a
  third, and `makeUncorsResponse`, `executeQuery`, `copyHeaders` disappear.
- Pairs naturally with a shared transport ([A09](A09-http-clients-are-rebuilt-per-mapping-and-per-reload.md))
  and with standard `http.Handler` middleware
  ([A05](A05-dual-handler-abstraction-and-unsafe-casts.md)).

## 5. Trade-offs and migration considerations

- **`ReverseProxy` requires an `http.Handler`.** It cannot be plugged into
  `contracts.Handler` without an adapter, so this change is most naturally
  sequenced *after* (or together with) [A05](A05-dual-handler-abstraction-and-unsafe-casts.md).
  An interim adapter is possible but adds another cast.
- **Behaviour will change in observable ways.** Notably: hop-by-hop headers stop
  being forwarded (correct, but a user relying on the current behaviour may
  notice), `X-Forwarded-For` starts being sent (some upstreams behave differently),
  and redirects: today `MakeHTTPClient` sets `CheckRedirect` to
  `http.ErrUseLastResponse` ([`internal/infra/httpclient.go:26`](../internal/infra/httpclient.go#L26))
  so 3xx are passed through with a rewritten `Location`. `ReverseProxy` does the
  same by default (it never follows redirects), so this behaviour is preserved —
  but the `Location` rewriting must move into `ModifyResponse`.
- **The rewrite-host feature must be ported.** `createReplacers`
  ([`internal/handler/proxy/handler.go:132`](../internal/handler/proxy/handler.go#L132))
  builds ad-hoc replacers from a context value; that logic moves into `Rewrite`
  unchanged, but note it currently constructs two `*Replacer` objects (each
  compiling a regexp) **per request** when a rewrite host is set — worth caching
  at config time as part of the same change.
- Keep the existing integration tests (`tests/integration/proxy`,
  `tests/integration/rewrite`, `tests/integration/domains`) as the acceptance
  gate; they cover the behaviour that must not regress.

## 6. Code references

| What | Where |
| --- | --- |
| Hand-rolled forwarding | [`internal/handler/proxy/handler.go:48`](../internal/handler/proxy/handler.go#L48) |
| Request construction | [`internal/handler/proxy/handler.go:75`](../internal/handler/proxy/handler.go#L75) |
| Response construction | [`internal/handler/proxy/handler.go:103`](../internal/handler/proxy/handler.go#L103) |
| Header copy (3 exclusions only) | [`internal/handler/proxy/helpers.go:15`](../internal/handler/proxy/helpers.go#L15) |
| Cookie rewriting | [`internal/handler/proxy/helpers.go:41`](../internal/handler/proxy/helpers.go#L41) |
| Per-request replacer construction | [`internal/handler/proxy/handler.go:132`](../internal/handler/proxy/handler.go#L132) |
| String-based URL replacement | [`internal/urlreplacer/replacer.go:88`](../internal/urlreplacer/replacer.go#L88) |
| `CheckRedirect` passthrough | [`internal/infra/httpclient.go:26`](../internal/infra/httpclient.go#L26) |
| Panicking close helper | [`internal/helpers/closer.go:6`](../internal/helpers/closer.go#L6) |
