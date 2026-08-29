# A14 — The 500 error page dumps a goroutine stack trace and process memory stats to the HTTP client

**Severity:** Medium-High
**Area:** Error handling / information exposure

---

## 1. What is wrong with the current approach

Every handler error is rendered by `infra.HTTPError`:

```go
// internal/infra/http_error.go:20-49
func HTTPError(writer http.ResponseWriter, err error) {
	... headers ...
	writer.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintln(writer, errorHeader)                       // ASCII-art "500 ERROR"
	fmt.Fprintf(writer, "An error occurred: %s\n", err)
	fmt.Fprint(writer, "Stack trace: ")
	fmt.Fprintln(writer, string(debug.Stack()))             // ← full goroutine stack
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)                          // ← stop-the-world
	fmt.Fprintln(writer, "Memory usage:")
	fmt.Fprintf(writer, "Alloc = %v\n", humanize.Bytes(memStats.Alloc))
	fmt.Fprintf(writer, "TotalAlloc = %v\n", humanize.Bytes(memStats.TotalAlloc))
	fmt.Fprintf(writer, "Sys = %v\n", humanize.Bytes(memStats.Sys))
	fmt.Fprintf(writer, "NumGC = %v\n", memStats.NumGC)
}
```

Four separate problems:

**(a) The stack trace is the wrong stack.** `debug.Stack()` captures the stack of
`HTTPError` itself, not of wherever the error originated. Since `HTTPError` is
called from `Server.handleRequest` ([`internal/server/server.go:205`](../internal/server/server.go#L205))
or from `CastToHTTPHandler` ([`internal/infra/handlers.go:32`](../internal/infra/handlers.go#L32)),
the trace is always the same handful of frames and contains no information about
the failure. It is pure noise that *looks* like a debugging aid.

**(b) It is sent to the HTTP client.** Go file paths, module paths, the local
username embedded in `GOPATH`-style paths, and process memory statistics are
written into a response body that a browser — and any page's JavaScript, since
the response carries `Access-Control-Allow-Origin: *`-style CORS headers from
upstream handlers — can read. uncors is explicitly documented as
development-only, so this is not a remote-attacker scenario; but any page loaded
through the proxy can trigger an error path and read the host's internal paths.

**(c) `runtime.ReadMemStats` is called on every error.** `ReadMemStats` stops the
world. Putting it on an error path that can be triggered by any request (a
missing upstream, an unmapped host — [`internal/handler/router/router.go:57`](../internal/handler/router/router.go#L57)
returns `errHostNotMapped` for *every* request to an unmapped host) means a
misconfigured browser extension hammering the proxy can repeatedly stop the world.

**(d) Every error is a 500.** An unreachable upstream (`dial tcp: connection
refused`), an unmapped host, a mock file that does not exist, and a Lua syntax
error all produce identical `500` responses. The correct codes are `502`, `404`,
`500` and `500` respectively; more importantly, the *message* the user needs
("host `foo.local` is not mapped — did you add it to your config?") is buried
under ASCII art and a stack trace.

Note also that the error text is **not** written to the log
([A13](A13-logging-and-output-are-three-parallel-systems.md)) — the response body is the
only place it appears. So the developer must inspect the failing HTTP response in
DevTools to learn what went wrong.

## 2. Why it is an architectural problem

- **The error boundary has no taxonomy.** There is one function that turns "any
  error" into "one response". Without typed errors, no call site can produce an
  appropriate status, an appropriate message, or an appropriate log level.
- **Diagnostics are delivered over the wrong channel.** The client is the wrong
  audience for a stack trace; the terminal and the log file are the right ones.
  Because logging is discarded by default ([A13](A13-logging-and-output-are-three-parallel-systems.md)),
  the response body became the de facto log — a workaround that hardened into a
  design.
- **A hot-path stop-the-world call in an error handler** is a denial-of-service
  amplifier that exists purely as debug scaffolding.

## 3. What the recommended approach is instead

**1. Introduce a small error taxonomy** and map it to status codes:

```go
// internal/infra
type HTTPStatusError struct{ Code int; Msg string; Err error }
func (e *HTTPStatusError) Error() string  { ... }
func (e *HTTPStatusError) Unwrap() error  { return e.Err }

func StatusFor(err error) (int, string) {
	var se *HTTPStatusError
	switch {
	case errors.As(err, &se):                    return se.Code, se.Msg
	case errors.Is(err, router.ErrHostNotMapped): return 404, "host is not mapped in the uncors config"
	case errors.Is(err, context.Canceled):        return 499, ""            // client gone: don't write
	case isUpstreamDialError(err):                return 502, "upstream is unreachable"
	default:                                      return 500, "internal error"
	}
}
```

**2. Log the detail, show the summary.**

```go
func HTTPError(w http.ResponseWriter, r *http.Request, err error) {
	code, msg := StatusFor(err)
	slog.Error("request failed", "method", r.Method, "url", r.URL.String(), "status", code, "err", err)
	if code == 499 { return }   // client disconnected; nothing to write
	writeErrorPage(w, code, msg, r)
}
```

The response page keeps the friendly uncors branding and gains something
genuinely useful: the request URL, the matched mapping (or "no mapping matched"),
and a one-line hint. It contains **no** stack trace and **no** memory statistics.

**3. Delete `runtime.ReadMemStats` from the error path.** Memory statistics
already have a proper home: the TUI's memory widget
([`internal/uncors_app/mem_widget.go`](../internal/uncors_app/mem_widget.go)), which
samples on a 2-second tick.

**4. Handle client cancellation explicitly.** The proxy handler already detects it
([`internal/handler/proxy/handler.go:34`](../internal/handler/proxy/handler.go#L34)) but
still returns the error, which then produces a 500 page written to a connection
that is already gone. Return a sentinel and short-circuit.

**5. Optionally gate verbose detail behind a flag.** If a stack trace is ever
genuinely wanted in the response, put it behind `--dev-errors` (off by default),
never on by default.

## 4. Why the proposed approach is better

- **Users get actionable errors.** "Host `api.local` is not mapped" with a 404 tells
  a developer exactly what to fix; a 500 with a stack trace of
  `infra.HTTPError` tells them nothing.
- **Correct status codes make the proxy honest to tooling.** Browsers, `curl`,
  fetch polyfills and test suites all behave differently on 404/502/500;
  collapsing everything to 500 breaks retry logic and error handling in the app
  being developed.
- **The stop-the-world call leaves the request path.**
- **Internal paths stop leaking into responses** that page JavaScript can read.
- **Errors reach the log**, which is where a developer looks — closing the loop
  with [A13](A13-logging-and-output-are-three-parallel-systems.md).

## 5. Trade-offs and migration considerations

- **Status codes will change**, which is user-visible: something currently
  observing 500 for an unreachable upstream will start seeing 502. This is a
  correctness fix but belongs in the release notes.
- **Snapshot tests will change.** `internal/infra/http_error_test.go` and any
  integration snapshots asserting the error body need regenerating.
- **Keep the ASCII-art branding if it is liked** — it costs nothing and makes the
  page recognisable; the objectionable parts are the trace and the mem stats.
- **`errHostNotMapped` is currently unexported and returned from a closure**
  ([`internal/handler/router/router.go:14`](../internal/handler/router/router.go#L14));
  export it (or expose a matcher) so `StatusFor` can recognise it.
- The `dustin/go-humanize` dependency is used only here and in the TUI mem widget;
  after this change it stays only for the widget.

## 6. Code references

| What | Where |
| --- | --- |
| Error page implementation | [`internal/infra/http_error.go:20`](../internal/infra/http_error.go#L20) |
| Wrong-stack trace | [`internal/infra/http_error.go:40`](../internal/infra/http_error.go#L40) |
| Stop-the-world in error path | [`internal/infra/http_error.go:42`](../internal/infra/http_error.go#L42) |
| Call site (server) | [`internal/server/server.go:205`](../internal/server/server.go#L205) |
| Call site (route cast) | [`internal/infra/handlers.go:32`](../internal/infra/handlers.go#L32) |
| Unmapped-host error → 500 | [`internal/handler/router/router.go:14`](../internal/handler/router/router.go#L14), [`:53`](../internal/handler/router/router.go#L53) |
| Client-cancellation detection | [`internal/handler/proxy/handler.go:34`](../internal/handler/proxy/handler.go#L34) |
| Memory widget (proper home for mem stats) | [`internal/uncors_app/mem_widget.go`](../internal/uncors_app/mem_widget.go) |
