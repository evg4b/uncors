# D07 — `ARCHITECTURE.md` and `CLAUDE.md` describe packages, types and a validation mechanism that no longer exist

**Severity:** Medium (misleads every new contributor and every AI-assisted change)
**Area:** Documentation vs implementation — internal docs

---

## 1. What is wrong

Both onboarding documents describe a previous shape of the codebase. Item by
item:

### Packages that do not exist

| Claim | Reality |
| --- | --- |
| `ARCHITECTURE.md`: "**Main Application (`internal/uncors`)** — Manages server lifecycle, graceful shutdown, and config watching" | No `internal/uncors` package. Its responsibilities are split between [`internal/cli`](../internal/cli/) and [`internal/uncors_app`](../internal/uncors_app/). |
| `CLAUDE.md`: "**`internal/uncors`** — Application lifecycle; `Uncors` type: manages server startup, graceful shutdown, and config watching" | Same. There is no `Uncors` type anywhere. |
| Both file trees list `internal/uncors/` | Neither lists the packages that actually exist: `internal/di`, `internal/cli`, `internal/uncors_app`, `internal/handler/router`, `pkg/urlt`. |
| `CLAUDE.md`: "`internal/server` — `RequestPrinter`: Goroutine that prints request info" | `RequestPrinter` exists but is never started — [A01](A01-request-tracker-deadlocks-headless-mode.md). |

### A validation mechanism that is not wired up

> **Configuration (`internal/config`)** — Loads and validates YAML config files
> using **JSON Schema**.
> — `ARCHITECTURE.md`

> `LoadConfiguration()`: Parses CLI flags and YAML config file. **JSON Schema
> validation (schema.json)**
> — `CLAUDE.md`

`LoadConfiguration` never touches `schema.json`
([`internal/config/config.go:26-66`](../internal/config/config.go#L26)); validation is
hand-written Go across `validate_primitives.go`, `validate_helpers.go` and the
per-type `Validate` methods. `schema.json`'s only reader in the entire repository
is [`tests/schema/schema_test.go`](../tests/schema/schema_test.go) — and it has drifted
from the real config ([D08](D08-schema-json-is-stale-and-unused.md)).

### Type signatures that were never in this codebase

```go
// ARCHITECTURE.md, "Key Design Patterns"
type Middleware = func(http.Handler) http.Handler
type ProxyHandlerFactory = func() contracts.Handler
```
```go
// CLAUDE.md, "Key Design Patterns"
type Middleware = func(http.Handler) http.Handler
```

The real definition is a three-argument interface
([`internal/contracts/http.go:30`](../internal/contracts/http.go#L30)):

```go
type Middleware interface {
	ServeHTTP(writer ResponseWriter, request *Request, next Next) error
}
```

`ProxyHandlerFactory` does not exist. Both "Extending UNCORS" sections then tell
contributors to "implement `func(http.Handler) http.Handler`" for a new
middleware — following that instruction produces something the router cannot
accept. (Ironically, the documented signature is the one this review recommends
adopting — [A05](A05-dual-handler-abstraction-and-unsafe-casts.md) — but it is not what
the code does today.)

Likewise, both documents instruct: "Add factory to **`RequestHandler`**" /
"Add factory method in **request handler routing**". There is no `RequestHandler`
type; new handlers are added to [`internal/di/public_api.go`](../internal/di/public_api.go)
and the `router.DI` interface ([`internal/handler/router/router.go:23`](../internal/handler/router/router.go#L23)).

### Lifecycle and request-flow claims

| Claim | Reality |
| --- | --- |
| `ARCHITECTURE.md`: HAR writers registered via `registerCloser`, closed "on shutdown **or config reload**" | No such symbol; closed only at process exit — [A02](A02-di-container-leaks-resources-on-every-config-reload.md), [D05](D05-har-docs-overstate-coverage-and-lifecycle.md) |
| `ARCHITECTURE.md` request flow: "HAR capture → options → cache → static" | Actual order is HAR → cache → OPTIONS → proxy, and static wraps that chain rather than sitting inside it ([`internal/handler/router/router.go:94`](../internal/handler/router/router.go#L94)) |
| `CLAUDE.md` request flow lists "HAR Collector (first middleware)" then "Options" then "Cache" then handler selection | Same ordering error, and it omits that mocks/scripts bypass all three ([A06](A06-cross-cutting-middleware-attached-to-one-branch-only.md)) |
| `ARCHITECTURE.md`: example CORS headers include `Access-Control-Allow-Methods: *` | Actual value is a fixed method list ([`internal/infra/cors.go:10`](../internal/infra/cors.go#L10)) — [D10](D10-mapping-and-cors-behaviour-does-not-match-docs.md) |

### Build and environment claims

| Claim | Reality |
| --- | --- |
| `CLAUDE.md`: "Language: Go **1.24.1+**" / "Requires Go 1.24.1+" | [`go.mod`](../go.mod) declares `go 1.26.4` |
| `CLAUDE.md`: "`make build` — Build binary" then "`./uncors --from …`" | `make build` runs `go build ./...`, which produces **no binary on disk** ([`Makefile`](../Makefile)). `make build-release` or `make install` is what produces one. |
| `CLAUDE.md`: "Integration Tests: in `tests/integration/` tagged with `// +build integration`" | The files use the modern `//go:build integration` form |
| `CLAUDE.md`: "`interactive`: Enable TUI mode (default: true)" listed under **config file** options | `Interactive` is `yaml:"-"` ([`internal/config/config.go:20`](../internal/config/config.go#L20)) — CLI-only, not settable in YAML |

## 2. Why it matters

These two files are the entry point for contributors and, given `CLAUDE.md`'s
purpose, the primary context for AI-assisted changes. Stale architecture
documentation is worse than none, because it is *specific*: a contributor
following "implement `func(http.Handler) http.Handler`" and "add a factory to
`RequestHandler`" will write code that does not compile, then have to reverse-engineer
the real design anyway — having first been actively misled about it.

The pattern is also diagnostic. Three of the errors (`internal/uncors`, JSON Schema
validation, `registerCloser`) describe a system that presumably *did* exist, which
means the docs were not updated alongside significant refactors. Without a
mechanism, they will drift again immediately after being fixed.

## 3. Recommended fix

**1. Rewrite both files against the current tree.** Minimum corrections:

- Replace `internal/uncors` with the real lifecycle owners
  (`internal/cli`, `internal/uncors_app`, `internal/di`), and add
  `internal/handler/router`, `internal/di`, `pkg/urlt` to both file trees.
- State plainly that config validation is hand-written Go and that `schema.json`
  is (currently) an editor-support artefact only — or wire the schema up
  ([D08](D08-schema-json-is-stale-and-unused.md)).
- Replace the invented type signatures with the real
  `contracts.Handler`/`contracts.Middleware`/`contracts.Next` definitions, and fix
  the "Extending UNCORS" steps to name `internal/di/public_api.go` and
  `router.DI`.
- Correct the middleware ordering, and state which routes the chain does and does
  not cover.
- Update the Go version, the `make build` description, and the build-tag syntax.
- Move `interactive` from the config-file list to the CLI list.

**2. Reduce the surface that can drift.** The most durable improvement is to stop
duplicating structural facts:

- Delete the ASCII file tree from both documents and replace it with a one-line
  `go list ./...` instruction, or generate it in `make format-docs`.
- Have `ARCHITECTURE.md` link to package doc comments rather than restate them;
  add real `// Package x …` comments (most packages have none) so `go doc` becomes
  the source of truth.
- `CLAUDE.md` currently duplicates ~80 % of `ARCHITECTURE.md`. Make `CLAUDE.md`
  short — conventions, commands, gotchas — and point to `ARCHITECTURE.md` for
  structure. One copy drifts half as fast as two.

**3. Add a cheap guard.** A test that asserts every directory named in the
documented file trees exists (and vice versa) is ~20 lines and would have caught
the `internal/uncors` reference the moment the package was removed.

## 4. Why this is better

- New contributors and AI agents get a correct model on the first read.
- Generated or linked structure cannot go stale silently.
- A single authoritative document plus package docs is less work to maintain than
  two overlapping hand-written trees.

## 5. Trade-offs and migration considerations

- **`CLAUDE.md` is consumed by tooling** and its "instructions override default
  behaviour" framing means errors in it propagate into generated code. Prioritise
  fixing the actionable claims (type signatures, extension steps, build commands)
  over the prose.
- **Auto-generating the file tree** requires the docs build to run in CI; if that
  is not wanted, the existence-check test in (3) is the lighter alternative.
- **Package doc comments are missing almost everywhere**; adding them is a
  separate, larger task, but it is the only version of this documentation that
  cannot drift from the code.
- Fix `ARCHITECTURE.md`, `CLAUDE.md`, and the `docs/` issues in this review
  together — several claims (HAR lifecycle, middleware order, CORS headers) appear
  in more than one file and should be corrected once, consistently.

## 6. Code and document references

| What | Where |
| --- | --- |
| Real config loading (no JSON Schema) | [`internal/config/config.go:26`](../internal/config/config.go#L26) |
| Hand-written validation | [`internal/config/validate_primitives.go`](../internal/config/validate_primitives.go), [`internal/config/validate_helpers.go`](../internal/config/validate_helpers.go) |
| Only reader of `schema.json` | [`tests/schema/schema_test.go`](../tests/schema/schema_test.go) |
| Real `Middleware` definition | [`internal/contracts/http.go:30`](../internal/contracts/http.go#L30) |
| Real extension point for handlers | [`internal/di/public_api.go`](../internal/di/public_api.go), [`internal/handler/router/router.go:23`](../internal/handler/router/router.go#L23) |
| Real middleware order | [`internal/handler/router/router.go:94`](../internal/handler/router/router.go#L94) |
| CORS method list | [`internal/infra/cors.go:10`](../internal/infra/cors.go#L10) |
| Go version | [`go.mod`](../go.mod) |
| `make build` target | [`Makefile`](../Makefile) |
| `Interactive` is CLI-only | [`internal/config/config.go:20`](../internal/config/config.go#L20) |
| Stale documents | [`ARCHITECTURE.md`](../ARCHITECTURE.md), [`CLAUDE.md`](../CLAUDE.md) |
