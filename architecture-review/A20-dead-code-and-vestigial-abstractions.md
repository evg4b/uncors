# A20 — Dead code and vestigial abstractions that misrepresent the design

**Severity:** Low-Medium (clarity, maintenance, and misleading signals)
**Area:** Whole tree

---

## 1. What is wrong with the current approach

A meaningful amount of the codebase describes capabilities that do not exist.
This is worth treating as an architectural issue rather than a cleanup chore,
because each item below actively misleads a reader about how the system works —
and in two cases the dead code marks a *hole* where working code should be.

| Symbol / artefact | Location | Status |
| --- | --- | --- |
| `server.RequestPrinter` | [`internal/server/request_printer.go:7`](../internal/server/request_printer.go#L7) | No call site. It is the missing consumer that causes the headless stall in [A01](A01-request-tracker-deadlocks-headless-mode.md). |
| `helpers.GracefulShutdown` / `waiteSignal` | [`internal/helpers/graceful_shutdown.go:18`](../internal/helpers/graceful_shutdown.go#L18) | No call site. Signal handling was re-implemented inline in [`internal/cli/run_non_ineractive.go:57`](../internal/cli/run_non_ineractive.go#L57) and omitted entirely from interactive mode ([A10](A10-interactive-and-headless-modes-are-two-parallel-implementations.md)). Note the typo `waiteSignal`. |
| `contracts.Logger` (10 methods) | [`internal/contracts/logger.go:3`](../internal/contracts/logger.go#L3) | Nothing implements or consumes it; only [`testing/mocks/logger_mock.go`](../testing/mocks/logger_mock.go) (2 400+ generated lines) references it. |
| `infra.MiddlewareFunc` | [`internal/infra/handlers.go:13`](../internal/infra/handlers.go#L13) | A third middleware shape, unused. See [A05](A05-dual-handler-abstraction-and-unsafe-casts.md). |
| `rewrite.IsRewriteRequest` | [`internal/handler/rewrite/helpers.go:26`](../internal/handler/rewrite/helpers.go#L26) | No call site. |
| `server.ErrNoCertificateForHost` | [`internal/server/errors.go:16`](../internal/server/errors.go#L16) | Declared, never returned. |
| `RistrettoCache.Wait()` | [`internal/handler/cache/cache.go:57`](../internal/handler/cache/cache.go#L57) | Public method used only by tests, while the same call is wrongly embedded in `Set` ([A16](A16-cache-design-issues.md)). |
| `static.Middleware.Wrap` | [`internal/handler/static/middleware.go:52`](../internal/handler/static/middleware.go#L52) | An adapter to a middleware shape nothing uses. |
| `cmd/diag/` | repository root | Empty committed directory. |
| Commented-out logging | [`internal/handler/router/router.go:54-56`](../internal/handler/router/router.go#L54) | Two dead `output.Errorf`/`log.Printf` lines where the unmapped-host diagnostic should be. |
| `uncors.log` (79 KB) | repository root | A committed runtime log file. |
| `schema.json` | repository root | Not used at runtime; only [`tests/schema/schema_test.go`](../tests/schema/schema_test.go) reads it, and it is out of sync with the config structs ([D08](D08-schema-json-is-stale-and-unused.md)). |

Two further vestigial *shapes* rather than symbols:

- **`di.factory1[T, D comparable]`** ([`internal/di/factory.go:24`](../internal/di/factory.go#L24))
  declares a key type parameter and then ignores it — it looks like a keyed cache
  and behaves like a singleton ([A02](A02-di-container-leaks-resources-on-every-config-reload.md)).
- **`helpers.AssertIsDefined`** ([`internal/helpers/asset.go:8`](../internal/helpers/asset.go#L8))
  uses `unsafe.Pointer` to inspect an interface's data word, replacing what should
  be a compile-time guarantee ([A04](A04-service-locator-di-container.md),
  [A12](A12-package-boundaries-and-layering-violations.md)). The file is named
  `asset.go` but contains an *assert*.

Naming defects in live code that compound the confusion: `infra.Mddleware` /
parameter `middlaware` ([`internal/infra/middleware.go:7`](../internal/infra/middleware.go#L7)),
`run_ineractive.go` / `runIneractive` and `run_non_ineractive.go` /
`runNonIneractive` ([`internal/cli/`](../internal/cli/)),
`router_helpres.go` ([`internal/handler/router/router_helpres.go`](../internal/handler/router/router_helpres.go)),
and the snake_case directory `internal/uncors_app` holding package `uncorsapp`.

## 2. Why it is an architectural problem

- **Dead code is read as documentation.** A reviewer who finds `RequestPrinter`
  reasonably concludes that headless mode prints requests. It does not — and the
  gap is a total service stall. `GracefulShutdown` tells the same lie about signal
  handling in the TUI.
- **Unused interfaces imply extension points that do not exist.**
  `contracts.Logger` and `infra.MiddlewareFunc` suggest a pluggable logging and
  middleware architecture; neither is real, and both compete with the mechanisms
  that are.
- **Generated mocks for dead interfaces inflate the tree** and consume review and
  regeneration effort forever.
- **Type signatures that lie** (`factory1`'s ignored key, `AssertIsDefined`'s
  unsafe pointer arithmetic) actively cause bugs, as [A02](A02-di-container-leaks-resources-on-every-config-reload.md)
  demonstrates.
- **A committed `uncors.log`** and an empty `cmd/diag/` are noise that suggests the
  repository hygiene is looser than it is.

## 3. What the recommended approach is instead

**Delete or complete — never leave in between.** Concretely:

1. **Complete** `RequestPrinter` by wiring it as the headless request sink
   ([A01](A01-request-tracker-deadlocks-headless-mode.md)) — this is the one item that must
   be *finished*, not removed.
2. **Complete** the unmapped-host diagnostic at
   [`internal/handler/router/router.go:54`](../internal/handler/router/router.go#L54)
   with a real logged message and a 404 ([A14](A14-http-error-page-leaks-stack-traces.md)).
3. **Delete**: `helpers.GracefulShutdown`, `contracts.Logger` +
   `testing/mocks/logger_mock.go`, `infra.MiddlewareFunc`,
   `rewrite.IsRewriteRequest`, `server.ErrNoCertificateForHost`,
   `static.Middleware.Wrap`, `cmd/diag/`, `uncors.log` (and add `*.log` to
   `.gitignore`).
4. **Replace** `factory1` with `sync.OnceValue`/a real keyed map, and
   `AssertIsDefined` with required constructor parameters.
5. **Decide** on `schema.json`: either wire it into `LoadConfiguration` as real
   runtime validation (which is what `ARCHITECTURE.md` and `CLAUDE.md` claim
   happens) and keep it generated from the Go structs, or demote it to an
   editor-support artefact, fix its drift, and correct the docs
   ([D08](D08-schema-json-is-stale-and-unused.md)).
6. **Fix the names** in one mechanical pass: `Mddleware`→`Middleware`,
   `run_ineractive`→`run_interactive`, `router_helpres`→`router_helpers`,
   `waiteSignal` (deleted with its file), `helpers/asset.go`→`helpers/assert.go`
   (or delete with the function), `internal/uncors_app`→`internal/tui/app`.

**Add a guard so this does not re-accumulate.** `golangci-lint` already runs with
`default: all`; `unused` is part of that set but does not flag exported symbols in
non-`main` packages. Since everything above lives under `internal/`, adding
[`unparam`](https://github.com/mvdan/unparam) and enabling
`unused.exported-fields`/a dead-code pass such as `deadcode`-style analysis over
`internal/...` would catch exported-but-unused declarations. Alternatively run
`go vet ./... && staticcheck -checks=U1000 ./internal/...` in CI.

## 4. Why the proposed approach is better

- **The code stops describing a system that does not exist.** Every remaining
  symbol is either used or absent, so reading the tree gives an accurate model.
- **The two "holes" get filled** rather than papered over — and one of them is the
  most severe defect found in this review.
- **~2 500 lines of generated mock plus a handful of files leave the repository**,
  reducing review surface and regeneration churn.
- **A CI check makes the property durable**, instead of relying on periodic manual
  audits.
- The naming pass removes small, constant friction: every reader of
  `infra.Mddleware` has to stop and check whether it is a typo or a different
  thing.

## 5. Trade-offs and migration considerations

- **Deleting `contracts.Logger` removes an extension point some future work might
  have wanted.** If a logging abstraction is wanted, `log/slog` is the answer
  ([A13](A13-logging-and-output-are-three-parallel-systems.md)), not a bespoke
  10-method interface — so deleting it is not losing an option.
- **Renaming files and packages produces a large, noisy diff** that will conflict
  with any in-flight branches. Do it as its own commit, ideally right before or
  after a release, and use `gopls rename` rather than `sed` for identifiers.
- **`uncors.log` may contain paths or hostnames from the author's machine**;
  removing it from the working tree does not remove it from git history. If that
  matters, it needs a history rewrite; otherwise deletion plus `.gitignore` is
  sufficient.
- **Enabling a stricter dead-code check may flag intentional API surface** in
  `pkg/urlt` (currently lint-excluded) and in `testing/`; scope the check to
  `internal/...` to avoid noise.

## 6. Code references

All references are inline in the table in section 1.
