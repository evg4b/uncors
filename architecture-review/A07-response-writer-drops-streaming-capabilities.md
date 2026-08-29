# A07 — `ResponseRecorder` drops `Flusher`/`Hijacker`, and body capture is unbounded

**Severity:** High
**Area:** HTTP core / proxying semantics

---

## 1. What is wrong with the current approach

Every response passes through `server.ResponseRecorder`, which **embeds the
`http.ResponseWriter` interface**:

```go
// internal/server/recorder.go:12-18
type ResponseRecorder struct {
	http.ResponseWriter
	statusCode int
	buf        *bytes.Buffer
	output     io.Writer
	startedAt  time.Time
}
```

Embedding an *interface* promotes only that interface's three methods
(`Header`, `Write`, `WriteHeader`). The optional interfaces that `net/http`'s
real response writer implements — `http.Flusher`, `http.Hijacker`,
`http.CloseNotifier`, `io.ReaderFrom` — are **not** promoted. A type assertion
such as `w.(http.Flusher)` on the recorder fails.

A grep confirms none of these interfaces are implemented or asserted anywhere in
the project:

```
$ grep -rn "Flusher\|Hijacker\|ReadFrom\|FlushInterval\|ReverseProxy" internal/
(no matches)
```

Consequences:

- **Server-Sent Events and other streamed responses do not stream.** `http.ServeContent`
  and the proxy's `io.Copy` write through the recorder into `net/http`'s internal
  2–4 KB `bufio.Writer`. Without an explicit `Flush`, a slow event stream sits in
  that buffer until it fills or the handler returns. An SSE endpoint proxied
  through uncors is effectively broken.
- **WebSocket and any `Connection: Upgrade` traffic is impossible.** `Hijacker` is
  unavailable, and the proxy handler uses `http.Client` anyway
  ([A08](A08-proxy-handler-reimplements-reverse-proxy.md)). An upgrade request gets
  a normal HTTP response and the connection dies.
- **`io.ReaderFrom` is lost**, so `io.Copy` cannot use the writer's fast path
  (`sendfile`/`writev`) for static files and proxied bodies; every byte round-trips
  through an intermediate buffer.

Separately, **body capture has no size limit**:

```go
// internal/server/recorder.go:44
func (r *ResponseRecorder) EnableBodyCapture() {
	if r.buf != nil { return }
	r.buf = &bytes.Buffer{}
	r.output = io.MultiWriter(r.buf, r.ResponseWriter)
}
```

Both the cache middleware
([`internal/handler/cache/middleware.go:63`](../internal/handler/cache/middleware.go#L63))
and the HAR middleware
([`internal/handler/har/middleware.go:60`](../internal/handler/har/middleware.go#L60))
call it unconditionally for matching requests. Proxying a 2 GB file download
through a mapping with `har:` enabled buffers the whole 2 GB in RAM — and then
`har.buildContent` copies it again into a `string`
([`internal/handler/har/content.go:44`](../internal/handler/har/content.go#L44)), and
the writer keeps it in `all []Entry` for the process lifetime
([`internal/handler/har/writer.go:32`](../internal/handler/har/writer.go#L32)).

The HAR middleware does the same to **request** bodies, with an extra copy:

```go
// internal/handler/har/middleware.go:46-54
var buf strings.Builder
n, _ := io.Copy(&buf, req.Body)     // whole request body into memory
req.Body = io.NopCloser(strings.NewReader(buf.String()))
```

Note also that the captured request bytes are counted but **never stored** in the
HAR entry (`BodySize` is set, `PostData` is not), so the buffering buys nothing.

## 2. Why it is an architectural problem

- **Wrapping `http.ResponseWriter` without preserving optional interfaces is a
  known Go hazard**, and here it is done at the outermost layer, so *no* handler in
  the system can stream. The limitation is invisible: nothing errors, responses
  merely arrive late or not at all.
- **Capture is an all-or-nothing switch on a shared object.** Because
  `EnableBodyCapture` is idempotent and lives on the writer, two middlewares that
  both want capture silently share one buffer, and neither can express "capture at
  most 1 MB" or "don't capture `Content-Type: video/*`".
- **Memory growth is a function of user traffic, not of configuration.** A dev
  tool that OOMs when someone downloads a large asset through it is failing at its
  one job. There is no cap, no eviction, and no way to opt out per response.

## 3. What the recommended approach is instead

**(a) Forward optional interfaces explicitly.**

```go
func (r *ResponseRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok { f.Flush() }
}
func (r *ResponseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := r.ResponseWriter.(http.Hijacker); ok { return h.Hijack() }
	return nil, nil, http.ErrNotSupported
}
func (r *ResponseRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter } // Go 1.20+ ResponseController
```

Implementing `Unwrap` makes the recorder cooperate with
`http.NewResponseController`, which is the modern, allocation-free way for
handlers to reach `Flush`/`SetWriteDeadline` through arbitrary wrappers.

Note that `Hijack` must **disable capture**: once hijacked there is no HTTP
response to record.

**(b) Cap and gate body capture.**

```go
type captureOpts struct {
	maxBytes  int64            // e.g. 5 MiB, configurable
	skip      func(http.Header) bool  // e.g. large or streaming content types
}
func (r *ResponseRecorder) EnableBodyCapture(o captureOpts)
```

Once `maxBytes` is exceeded, stop appending and mark the capture truncated;
`ResponseCapture` gains a `Truncated bool` so the HAR entry can honestly record
`"comment": "body truncated"` and the cache can decline to store the entry
(caching a truncated body would be a correctness bug — today an over-large body is
cached in full, which is merely expensive, but the cap must not change that into
storing a partial body).

Also **skip capture when the response is streaming**: `Content-Type:
text/event-stream`, `Transfer-Encoding: chunked` with no `Content-Length`, or a
`Content-Length` above the cap known up front.

**(c) Stop buffering request bodies in HAR until they are actually recorded.**
Either record `PostData` (and cap it the same way) or drop the buffering entirely
and report `BodySize` from `Content-Length`. The current code pays the full cost
of buffering and stores none of it.

**(d) Add a flush interval for proxied responses.** With `Flush` available, the
proxy can flush periodically (`httputil.ReverseProxy` does this via
`FlushInterval`, another reason to adopt it — see
[A08](A08-proxy-handler-reimplements-reverse-proxy.md)).

## 4. Why the proposed approach is better

- **SSE, long-polling and streamed JSON work**, which are ordinary things for the
  APIs a dev proxy sits in front of.
- **WebSocket upgrades become possible** once `Hijack` is available and the proxy
  is switched to a hijack-capable implementation.
- **Memory becomes bounded by configuration rather than by whatever the user
  downloads**, removing an OOM class entirely.
- **`ResponseController` support future-proofs the wrapper**: any future
  middleware that wraps the writer keeps working as long as it implements
  `Unwrap`.
- **Honest HAR output.** A truncated body flagged as truncated is more useful than
  a 2 GB HAR file or a silent OOM.

## 5. Trade-offs and migration considerations

- **Adding `Flush()` changes response framing.** Once the recorder is a `Flusher`,
  `http.ServeContent` and others may flush earlier, producing chunked responses
  where a buffered `Content-Length` response was produced before. This is correct
  behaviour but will change existing snapshot tests in
  `tests/integration/**/__snapshots__`.
- **A capture cap changes cache behaviour for large bodies.** Decide deliberately:
  recommended is "responses above the cap are not cached", which is what most HTTP
  caches do.
- **`Hijack` support means the recorder can no longer assume it owns the
  connection.** Guard `Captured()` so it reports nothing after a hijack.
- The changes are local to `internal/server/recorder.go` plus the two capture call
  sites, and are worth doing before the wider handler refactor in
  [A05](A05-dual-handler-abstraction-and-unsafe-casts.md) since they are independent.

## 6. Code references

| What | Where |
| --- | --- |
| Interface embedding drops optional interfaces | [`internal/server/recorder.go:12`](../internal/server/recorder.go#L12) |
| Unbounded capture | [`internal/server/recorder.go:44`](../internal/server/recorder.go#L44) |
| Cache enables capture | [`internal/handler/cache/middleware.go:63`](../internal/handler/cache/middleware.go#L63) |
| HAR enables capture | [`internal/handler/har/middleware.go:60`](../internal/handler/har/middleware.go#L60) |
| HAR buffers request body and discards it | [`internal/handler/har/middleware.go:46`](../internal/handler/har/middleware.go#L46) |
| Body copied again into a string | [`internal/handler/har/content.go:44`](../internal/handler/har/content.go#L44) |
| Entries retained for process lifetime | [`internal/handler/har/writer.go:32`](../internal/handler/har/writer.go#L32) |
| Proxy body copy with no flushing | [`internal/handler/proxy/handler.go:124`](../internal/handler/proxy/handler.go#L124) |
| Recorder installed for every request | [`internal/server/server.go:183`](../internal/server/server.go#L183) |
