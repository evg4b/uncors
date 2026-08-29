# UNCORS Architecture

How UNCORS works and how the code is organised. For the user-facing reference see
[docs/](docs/); for contributor conventions see [CONTRIBUTING.md](CONTRIBUTING.md).

## What is UNCORS?

UNCORS is a local development proxy that bypasses CORS restrictions. It sits
between your browser and your backends, rewriting hosts and CORS headers on the
fly, and can answer requests locally from mocks, Lua scripts or static files.

## Layers

The dependency direction is one-way. Nothing on a lower line may import
something from a higher one.

```
internal/cli                     command tree, run modes  (composition root)
internal/uncors, internal/di     application lifecycle, per-generation wiring
internal/server, internal/tui    listeners and TLS, terminal rendering
internal/handler/*               handlers and middleware
internal/config, internal/infra  configuration, shared HTTP infrastructure
internal/contracts               the few interfaces the layers share
```

Run `go list ./...` for the current package list, and `go doc ./internal/<pkg>`
for what a package is for — the package comments are the source of truth, not
this file.

## Core components

**`internal/cli`** — the command tree. It resolves the command named on the
command line, parses its flags, and runs it. `uncors` itself is the root command;
`generate-certs` is a subcommand. This is the only package that decides between
the terminal UI and plain output.

**`internal/uncors`** — the application. `Uncors` owns the server and the current
configuration generation; `Runner` owns the process lifecycle (startup, config
watching, the version check, signal handling, shutdown); `Reloader` owns config
hot reload. Both run modes drive the same three types, so their behaviour cannot
diverge.

**`internal/di`** — the composition root. `Container` holds the process-scoped
values (filesystem, output, version) and the application-scoped singletons
(server, request tracker, certificate manager, HTTP client pool). `Runtime` holds
everything derived from *one* configuration — routers, HAR writers, the response
cache, the server targets — and releases exactly that on `Close`, which is what
makes a reload leak-free.

**`internal/config`** — loading and validation. Validation is hand-written Go:
each type has a `Validate(field, …)` method, and the errors are joined with
dotted field paths. `schema.json` is an editor-support artefact for YAML
completion; it is not consulted at runtime.

**`internal/handler`** — the request graph.

| Handler  | Answers with                              |
| -------- | ----------------------------------------- |
| `proxy`  | the upstream, via `httputil.ReverseProxy` |
| `mock`   | a configured response                     |
| `script` | the result of a compiled Lua script       |
| `static` | a file from disk                          |

| Middleware | Does                                              |
| ---------- | ------------------------------------------------- |
| `har`      | records the traffic of a mapping to a HAR archive |
| `options`  | answers CORS preflights                           |
| `cache`    | serves repeated upstream responses from memory    |
| `rewrite`  | rewrites a path and re-enters routing             |

`router` assembles them into one router per mapping.

**`internal/server`** — TCP and TLS listeners, per-host certificate generation
with an SNI cache, and the request activity tracker.

**`internal/infra`** — shared HTTP infrastructure: the client pool, the response
recorder, CORS headers, the error page, and logging setup.

**`internal/tui`** — console rendering, with `internal/tui/app` holding the
BubbleTea model.

## Request flow

1. A **listener** accepts the request and wraps the writer in an
   `infra.ResponseRecorder`, which observes the status and, on request, the body.
   An activity event is emitted for the tracker.
2. The **host router** picks the mapping whose `from:` matches.
3. **Mapping-wide middleware** runs, outermost first: `har` (so every response
   the mapping produces is recorded) then `options` (so a preflight is answered
   before anything decides what serves the path).
4. The **mapping's routes** are tried in precedence order: mocks, then scripts,
   then rewrites, then static mounts, then the proxy. A rewritten request
   re-enters this list, bounded at 8 rounds.
5. The **cache** wraps the proxy branch only: a mock, a script or a static file
   is produced locally and changes when the config does, so caching it would only
   serve a stale copy.
6. **CORS headers** are written by whichever handler answered.
7. The recorder reports the final status, and a terminal activity event is
   emitted.

The headers added to a proxied response are:

```
Access-Control-Allow-Origin: <the request's Origin, or * when it had none>
Access-Control-Allow-Credentials: true
Access-Control-Allow-Headers: *
Access-Control-Allow-Methods: GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, LINK, OPTIONS
Access-Control-Expose-Headers: *
Access-Control-Max-Age: 86400
```

A preflight answered by the `options` middleware echoes the request's
`Access-Control-Request-Headers` and `-Method` instead of the wildcards.

## Key design patterns

**Standard net/http shapes.** Handlers are `http.Handler` and middleware is
`func(http.Handler) http.Handler`:

```go
// internal/contracts
type Middleware = func(http.Handler) http.Handler
```

Handlers that benefit from returning an error opt in locally; `infra.HandlerFunc`
renders a returned error as an HTTP error response with an appropriate status.

**Explicit dependencies.** The router states what it needs as a value the
composition root fills in, rather than reaching back into a container:

```go
// internal/handler/router
type Deps struct {
    Proxy  http.Handler
    Mock   func(*config.Response) http.Handler
    Static func(path string, dir config.StaticDirectory) contracts.Middleware
    // …
}
```

**Functional options.** Every handler and middleware is constructed with
`WithX(...)` options.

**Generation-scoped resources.** Anything derived from the configuration lives in
`di.Runtime` and is closed when that configuration stops being current.

## Extending UNCORS

**A new handler:**

1. Create `internal/handler/<name>/`, implementing `http.Handler`.
2. Add a constructor to `internal/di/public_api.go`.
3. Add a field for it to `router.Deps` and fill it in `internal/di/runtime.go`.
4. Register its routes in `router.registerMapping`, in precedence order.

**A new middleware:**

1. Create `internal/handler/<name>/` with a `Wrap(http.Handler) http.Handler`
   method.
2. Decide whether it belongs to the whole mapping (like `har`) or to one branch
   (like `cache`) and wire it in `router.registerMapping` accordingly.

**A new config option:**

1. Add the field and its `Validate` rules in `internal/config/`.
2. Update `schema.json` for editor completion.
3. Document it — `tests/docs` loads every example in `docs/`, and
   `internal/cli` pins the documented flag table to the real flag set.

## Testing

- Unit tests live beside the code; mocks are generated with `minimock`.
- `tests/integration` boots the real proxy over real sockets and TLS
  (`make test-integration`).
- `tests/docs` loads every configuration example in the documentation.
- `make dead-code` reports anything under `internal/` that nothing can reach.

## Important notes

**Security.** UNCORS is a development tool: it strips CORS protections and can
hold a locally trusted CA. It binds to loopback by default; `--listen` opts out
of that, loudly.

**Performance.** One shared HTTP transport per upstream proxy setting, an
in-memory response cache, non-blocking activity reporting, and HAR recording
through an append-only journal.
