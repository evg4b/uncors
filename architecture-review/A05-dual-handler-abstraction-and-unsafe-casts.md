# A05 — Two competing handler abstractions, bridged by a cast that panics

**Severity:** High
**Area:** HTTP core abstractions

---

## 1. What is wrong with the current approach

UNCORS defines its own handler and middleware types that shadow the standard
library's:

```go
// internal/contracts/http.go:23-33
type Handler interface {
	ServeHTTP(writer ResponseWriter, request *Request) error
}
type Next func(writer ResponseWriter, request *Request) error
type Middleware interface {
	ServeHTTP(writer ResponseWriter, request *Request, next Next) error
}
```

`contracts.ResponseWriter` is a superset of `http.ResponseWriter`
([`internal/contracts/response_writer.go:22`](../internal/contracts/response_writer.go#L22)).
But routing uses `gorilla/mux`, which only speaks `http.Handler`. So the codebase
converts back and forth on every route:

```go
// internal/infra/handlers.go:25
func CastToHTTPHandler(handler contracts.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		writer, ok := response.(contracts.ResponseWriter)
		if !ok {
			panic(ErrResponseNotCasted)      // ← runtime type assertion in the hot path
		}
		...
	})
}

// internal/infra/handlers.go:38
func CastToContractsHandler(handler http.Handler) contracts.Handler
```

The round trip appears at least twice per request:
`di.Router` wraps the mux router with `CastToContractsHandler`
([`internal/di/public_api.go:169`](../internal/di/public_api.go#L169)), and every
route inside wraps its handler with `CastToHTTPHandler`
([`internal/handler/router/router_helpres.go:52`](../internal/handler/router/router_helpres.go#L52),
[`internal/handler/router/router.go:17`](../internal/handler/router/router.go#L17)).

The whole scheme only works because `Server.handleRequest` wraps the writer in a
`*ResponseRecorder` before the router ever sees it
([`internal/server/server.go:183`](../internal/server/server.go#L183)). That is an
invisible, undocumented contract: **anything that ever wraps the
`http.ResponseWriter` between the server and a route — a stdlib
`http.TimeoutHandler`, a compression middleware, an HTTP/2 push wrapper, a future
`h2c` upgrade — turns every request into a panic.**

There is also a third middleware shape in the same file that nothing uses:

```go
// internal/infra/handlers.go:13
type MiddlewareFunc func(contracts.Handler) contracts.Handler
```

and a fourth in `ARCHITECTURE.md`, which documents `type Middleware =
func(http.Handler) http.Handler` — a signature that does not exist anywhere in
the tree (see [D07](D07-architecture-and-claude-md-are-stale.md)).

Finally, the composition helper is misspelled and untyped:

```go
// internal/infra/middleware.go:7
func Mddleware(middlaware contracts.Middleware, handler contracts.Handler) contracts.Handler
```

`Mddleware` / `middlaware` appear at six call sites in the router.

## 2. Why it is an architectural problem

- **A runtime `panic` is used to enforce a type invariant that the design cannot
  express.** The compiler cannot check "the writer at this point is always a
  `*ResponseRecorder`", so the code asserts and panics. The correct response to
  "my abstraction cannot be expressed in the type system" is to change the
  abstraction, not to add an assertion.
- **The custom types buy little.** The two additions over `http.Handler` are
  (a) an `error` return, and (b) body capture. Both have idiomatic Go solutions
  that do not require forking the interface (see below). Meanwhile the fork costs
  every route two adapter allocations, blocks the use of the entire ecosystem of
  `http.Handler` middleware, and forces the panic above.
- **The abstraction is leaky in both directions.** Handlers routinely down-convert
  their parameters back to the stdlib types anyway —
  `func (h *Handler) handle(resp http.ResponseWriter, req *http.Request)` in
  [`internal/handler/proxy/handler.go:48`](../internal/handler/proxy/handler.go#L48),
  `serveRawContent(writer http.ResponseWriter)` in
  [`internal/handler/mock/handler.go:79`](../internal/handler/mock/handler.go#L79),
  `handle(resp http.ResponseWriter, req *http.Request)` in
  [`internal/handler/options/middleware.go:34`](../internal/handler/options/middleware.go#L34).
  When the interior of nearly every handler works in `http.ResponseWriter`, the
  wrapper type is ceremony.
- **Error returns are not actually used as errors.** `CastToHTTPHandler` converts a
  returned error into `infra.HTTPError`, which writes a 500 page. So does
  `Server.handleRequest` ([`internal/server/server.go:205`](../internal/server/server.go#L205)).
  Handlers *also* log the error themselves before returning it
  ([`internal/handler/proxy/handler.go:38`](../internal/handler/proxy/handler.go#L38),
  [`internal/handler/mock/handler.go:41`](../internal/handler/mock/handler.go#L41)).
  The net effect of the `error` return is "write a 500", which a plain
  `http.Handler` can do directly.

## 3. What the recommended approach is instead

**Standardise on `http.Handler` and `func(http.Handler) http.Handler`.**

1. **Body capture becomes a writer wrapper, not an interface in `contracts`.**
   Keep `*server.ResponseRecorder`, but have consumers obtain it through a
   *checked helper* rather than a bare assertion:

```go
// returns (capture, true) only if the pipeline installed a recorder
func CaptureFrom(w http.ResponseWriter) (*ResponseRecorder, bool)
```

   Middleware that needs capture (HAR, cache) calls it and degrades gracefully —
   e.g. HAR records the entry without a body — instead of panicking the process.
   Better still: install the recorder *inside* the HAR/cache middleware, so the
   need is local and the dependency is explicit.

2. **Errors become an explicit helper, not an interface change.**

```go
type errHandler func(http.ResponseWriter, *http.Request) error

func handle(h errHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil { infra.HTTPError(w, r, err) }
	})
}
```

   Individual handlers opt in where they benefit; the pipeline stays stdlib.

3. **Middleware becomes the conventional shape**, which is what
   `gorilla/mux.Router.Use` already expects:

```go
type Middleware = func(http.Handler) http.Handler
```

   `infra.Mddleware` disappears; composition is `mw1(mw2(h))` or a tiny
   `chain(h, mws...)`.

4. **Delete `CastToHTTPHandler` / `CastToContractsHandler` / `MiddlewareFunc` /
   `contracts.Handler` / `contracts.Middleware` / `contracts.Next`.**
   Keep `contracts.ResponseCapture` as a plain data struct.

## 4. Why the proposed approach is better

- **The panic disappears** because there is no longer a type that "must" be
  present at a boundary the compiler cannot see.
- **The whole `net/http` ecosystem becomes usable**: `http.TimeoutHandler`,
  `httputil.ReverseProxy` (see [A08](A08-proxy-handler-reimplements-reverse-proxy.md)),
  `h2c`, `gziphandler`, `otelhttp`, `mux.Router.Use` — none of which can be
  dropped into the current pipeline.
- **Two adapter closures per route per request are removed** from the hot path,
  along with the interface conversions they carry.
- **New contributors read a familiar shape.** `func(http.Handler) http.Handler` is
  the single most recognisable pattern in Go web code; a bespoke three-argument
  `ServeHTTP(w, r, next) error` is not.
- The stale documentation in `ARCHITECTURE.md` and `CLAUDE.md` becomes correct
  by accident — both already describe the standard signature.

## 5. Trade-offs and migration considerations

- **This touches every handler and middleware package.** It is mechanical but
  broad. Suggested order:
  1. Change `contracts.Middleware` to `func(http.Handler) http.Handler` and add
     `infra.chain`; adapt the six `infra.Mddleware` call sites in the router.
  2. Convert middlewares one at a time (`options` → `cache` → `har` → `rewrite` →
     `static`); each is independently shippable because the router can adapt.
  3. Convert handlers (`mock`, `script`, `proxy`) — mostly deleting the
     `contracts.` prefix, since their bodies already use stdlib types.
  4. Delete the casts and the `contracts` handler types last.
- **`error`-returning handlers are genuinely nicer to write.** Keep the
  `errHandler` helper so the ergonomics are not lost; the point is that it should
  be a local convenience, not a global interface.
- **Body capture must be audited during the move.** Today `EnableBodyCapture` is
  idempotent and shared between HAR and cache
  ([`internal/server/recorder.go:44`](../internal/server/recorder.go#L44)); if the
  recorder moves into the middlewares, make sure two nested capturers don't
  double-buffer the same body (wrap once, share the buffer via context or a
  single recorder installed by the outermost capturer).
- Fix the `Mddleware`/`middlaware` spelling as part of the change rather than
  separately, to avoid two churn passes over the same lines.

## 6. Code references

| What | Where |
| --- | --- |
| Custom handler/middleware types | [`internal/contracts/http.go:26`](../internal/contracts/http.go#L26) |
| Custom response writer | [`internal/contracts/response_writer.go:22`](../internal/contracts/response_writer.go#L22) |
| Panicking cast | [`internal/infra/handlers.go:25`](../internal/infra/handlers.go#L25) |
| Reverse cast | [`internal/infra/handlers.go:39`](../internal/infra/handlers.go#L39) |
| Unused third middleware shape | [`internal/infra/handlers.go:13`](../internal/infra/handlers.go#L13) |
| Misspelled composer | [`internal/infra/middleware.go:7`](../internal/infra/middleware.go#L7) |
| Recorder installed by the server | [`internal/server/server.go:183`](../internal/server/server.go#L183) |
| Casts on every route | [`internal/handler/router/router_helpres.go:52`](../internal/handler/router/router_helpres.go#L52), [`internal/handler/router/router.go:17`](../internal/handler/router/router.go#L17), [`internal/di/public_api.go:169`](../internal/di/public_api.go#L169) |
| Handlers that immediately down-convert | [`internal/handler/proxy/handler.go:48`](../internal/handler/proxy/handler.go#L48), [`internal/handler/mock/handler.go:79`](../internal/handler/mock/handler.go#L79), [`internal/handler/options/middleware.go:34`](../internal/handler/options/middleware.go#L34) |
