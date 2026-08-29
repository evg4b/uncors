# UNCORS — Architecture & Documentation Review

A structural review of the UNCORS codebase (`main`-line, branch `review`,
commit `adeac37`) from the perspective of building and maintaining a local
CLI/HTTP proxy tool.

Scope: architectural concerns only — package boundaries and responsibilities,
lifecycle and hot reload, CLI and run-mode structure, HTTP/proxy abstractions,
and performance changes that require design changes rather than micro-optimisation.
Plus a comparison of everything under `docs/` (and `ARCHITECTURE.md` /
`CLAUDE.md`) against what the code actually does.

Every report follows the same structure: what is wrong, why it is architectural,
what to do instead, why that is better, trade-offs and migration notes, and
direct links to the code involved.

---

## How to read this

Findings marked **verified** were reproduced with a throwaway test against the
current tree; the reproduction is quoted in the report. Everything else is
established by direct code reading with line references.

---

## Architecture findings

| # | Finding | Severity | Theme |
| --- | --- | --- | --- |
| [A01](A01-request-tracker-deadlocks-headless-mode.md) | Request-tracker channel has no consumer in headless mode; **the proxy permanently stalls after ~333 requests** *(verified)* | **Critical** | Server core |
| [A02](A02-di-container-leaks-resources-on-every-config-reload.md) | Every config reload leaks a HAR writer + goroutine and can corrupt the HAR file; `cache-config` changes never apply | **Critical** | Lifecycle |
| [A03](A03-hot-reload-is-implemented-twice-and-unsafely.md) | Hot reload implemented twice; one copy discards config errors and restarts with a `nil` config; reload is destructive on failure | High | Lifecycle |
| [A04](A04-service-locator-di-container.md) | `di.Container` is a service locator that the router reaches back into | High | DI / structure |
| [A05](A05-dual-handler-abstraction-and-unsafe-casts.md) | Two competing handler abstractions bridged by a cast that `panic`s | High | HTTP core |
| [A06](A06-cross-cutting-middleware-attached-to-one-branch-only.md) | HAR / cache / OPTIONS wrap only the proxy branch; mocks, scripts and statics bypass them; route precedence is accidental *(verified)* | High | Routing |
| [A07](A07-response-writer-drops-streaming-capabilities.md) | `ResponseRecorder` drops `Flusher`/`Hijacker` (no SSE, no WebSocket) and captures bodies without a size limit | High | HTTP core |
| [A08](A08-proxy-handler-reimplements-reverse-proxy.md) | The proxy hand-rolls forwarding instead of using `httputil.ReverseProxy`; hop-by-hop headers forwarded, no `X-Forwarded-For`, no streaming | High | Proxy |
| [A09](A09-http-clients-are-rebuilt-per-mapping-and-per-reload.md) | A new `http.Transport` per port group and per reload; `MaxIdleConnsPerHost` left at 2; none ever closed | Medium-High | Performance |
| [A10](A10-interactive-and-headless-modes-are-two-parallel-implementations.md) | Interactive and headless modes are two independent implementations that have already diverged (signals, timeouts, request display) | High | App structure |
| [A11](A11-cli-command-structure.md) | Subcommand dispatch hand-rolled in `main`, split across two packages, with two flag sets and a hidden `--version` | Medium | CLI |
| [A12](A12-package-boundaries-and-layering-violations.md) | `config` imports `server` and `tui`; `helpers` is a junk drawer containing `unsafe` pointer arithmetic | Medium | Structure |
| [A13](A13-logging-and-output-are-three-parallel-systems.md) | Three parallel output systems; **all internal diagnostics discarded by default**, including `net/http` server errors | High | Observability |
| [A14](A14-http-error-page-leaks-stack-traces.md) | The 500 page ships a stack trace and `runtime.ReadMemStats` to the HTTP client; every error is a 500 | Medium-High | Error handling |
| [A15](A15-har-writer-rewrites-the-whole-archive-on-every-batch.md) | HAR writer re-serialises and rewrites the entire archive per batch — O(N²) I/O, unbounded memory | Medium-High | Performance |
| [A16](A16-cache-design-issues.md) | `ristretto.Wait()` blocks every cached response; cache key ignores request body, `Vary` and port | Medium-High | Caching |
| [A17](A17-pkg-urlt-is-a-fork-of-net-url.md) | `pkg/urlt` is a ~1 500-line fork of `net/url` maintained to allow `{}` in hostnames | Medium | URL handling |
| [A18](A18-listen-address-is-hardcoded-to-loopback.md) | Listen address hard-coded to `127.0.0.1`; no `--listen`; breaks Docker and device testing | Medium | Config surface |
| [A19](A19-lua-script-handler-lifecycle.md) | Lua scripts re-read and recompiled per request; no context, no timeout, undeclared sandbox | Medium | Script handler |
| [A20](A20-dead-code-and-vestigial-abstractions.md) | Dead code and type signatures that misrepresent the design (and mark two real holes) | Low-Medium | Whole tree |

## Documentation findings

| # | Finding | Severity |
| --- | --- | --- |
| [D01](D01-debug-flag-does-not-exist.md) | `--debug` and `debug:` are documented 10× and do not exist; `--interactive` and `--version` are undocumented | **High** |
| [D02](D02-durations-with-spaces-do-not-parse.md) | Every documented multi-unit duration (`1m 30s`, `1h 30m`, `2s 500ms`) fails to parse and aborts startup *(verified)* | **High** |
| [D03](D03-tilde-paths-are-not-expanded.md) | `~/…` paths are used throughout the docs and are never expanded | Medium |
| [D04](D04-spa-with-mocks-example-is-broken.md) | The flagship "SPA with API Proxying" example is broken — the static mount shadows every mock *(verified)* | **High** |
| [D05](D05-har-docs-overstate-coverage-and-lifecycle.md) | HAR docs claim it records everything and closes on reload; neither is true | Medium-High |
| [D06](D06-rewrite-docs-do-not-match-behaviour.md) | Rewrite docs promise scheme upgrades and rewrite→mock chaining; neither happens | Medium |
| [D07](D07-architecture-and-claude-md-are-stale.md) | `ARCHITECTURE.md` / `CLAUDE.md` describe packages, types and a validation mechanism that do not exist | Medium |
| [D08](D08-schema-json-is-stale-and-unused.md) | `schema.json` is unused at runtime and has drifted in four places | Medium |
| [D09](D09-docker-instructions-cannot-work.md) | The documented `docker run` command cannot work for four independent reasons | **High** |
| [D10](D10-mapping-and-cors-behaviour-does-not-match-docs.md) | Assorted verified mismatches: `proxy:` example, `//` scheme mappings, CORS header set, placeholder matching, POST caching | Medium |

---

## Suggested sequencing

The findings are heavily interconnected. A workable order, front-loading the
changes that fix real defects at low risk:

**Phase 1 — stop the bleeding (small, isolated, high value)**

1. [A01](A01-request-tracker-deadlocks-headless-mode.md) — non-blocking `Emit` + a real headless request sink. *(fixes a total service stall)*
2. [A13](A13-logging-and-output-are-three-parallel-systems.md) — default logging to stderr, wire `http.Server.ErrorLog`. *(makes everything else diagnosable)*
3. [A16](A16-cache-design-issues.md) step 1 — remove `ristretto.Wait()` from `Set`. *(one line)*
4. [A09](A09-http-clients-are-rebuilt-per-mapping-and-per-reload.md) — shared, tuned transport. *(largest single perf win)*
5. [A14](A14-http-error-page-leaks-stack-traces.md) — drop the stack trace and `ReadMemStats`, add a status taxonomy.
6. [A07](A07-response-writer-drops-streaming-capabilities.md) — forward `Flusher`/`Hijacker`/`Unwrap`, cap body capture.
7. [D01](D01-debug-flag-does-not-exist.md)–[D04](D04-spa-with-mocks-example-is-broken.md), [D10](D10-mapping-and-cors-behaviour-does-not-match-docs.md) — documentation corrections, plus the docs-example test from [D02](D02-durations-with-spaces-do-not-parse.md) §5.

**Phase 2 — lifecycle**

8. [A02](A02-di-container-leaks-resources-on-every-config-reload.md) — generation-scoped resources with real teardown.
9. [A03](A03-hot-reload-is-implemented-twice-and-unsafely.md) + [A10](A10-interactive-and-headless-modes-are-two-parallel-implementations.md) — one `Runtime`, two presenters.
10. [A11](A11-cli-command-structure.md) — one command tree; separate flag parsing from config loading.
11. [A18](A18-listen-address-is-hardcoded-to-loopback.md) + [D09](D09-docker-instructions-cannot-work.md) — `--listen`, `--ca-dir`, working container image.

**Phase 3 — HTTP core**

12. [A05](A05-dual-handler-abstraction-and-unsafe-casts.md) — standardise on `http.Handler`.
13. [A06](A06-cross-cutting-middleware-attached-to-one-branch-only.md) + [D04](D04-spa-with-mocks-example-is-broken.md) + [D05](D05-har-docs-overstate-coverage-and-lifecycle.md) — wrap the mapping, define route precedence.
14. [A08](A08-proxy-handler-reimplements-reverse-proxy.md) + [A17](A17-pkg-urlt-is-a-fork-of-net-url.md) — `httputil.ReverseProxy` with a structural URL replacer.

**Phase 4 — cleanup**

15. [A04](A04-service-locator-di-container.md) — replace the locator with explicit wiring.
16. [A12](A12-package-boundaries-and-layering-violations.md) — fix layering, dissolve `helpers`.
17. [A15](A15-har-writer-rewrites-the-whole-archive-on-every-batch.md), [A16](A16-cache-design-issues.md) (remainder), [A19](A19-lua-script-handler-lifecycle.md), [A20](A20-dead-code-and-vestigial-abstractions.md), [D07](D07-architecture-and-claude-md-are-stale.md), [D08](D08-schema-json-is-stale-and-unused.md).

---

## What the review found healthy

Not everything needs changing, and several decisions are worth preserving through
any refactor:

- **The config type design.** Per-type `Validate(field, fs)` with `errors.Join`
  and dotted field paths produces genuinely good error messages, and the
  shorthand `UnmarshalYAML` forms (`from: to`, scalar `har:`, map `statics:`) are
  a real ergonomic win.
- **The integration test harness** (`testing/integration`) — real sockets, real
  TLS, in-memory host resolution, a recording backend — is well built and is the
  right foundation for the regression tests several of these findings need.
- **The functional-options constructors** are consistent across every handler and
  middleware, which makes those packages easy to read and to test.
- **Per-host on-the-fly certificate generation** with an SNI-driven cache
  (`internal/server/host_cert_manager.go`) is a good design and the right
  replacement for the shared-certificate model the Migration Guide describes.
- **The HAR collector's channel-based non-blocking capture** is the correct shape;
  it is the *persistence* strategy behind it that needs work ([A15](A15-har-writer-rewrites-the-whole-archive-on-every-batch.md)).
- **Documentation coverage is broad and well written.** The problems catalogued
  here are drift, not absence — which is a much better starting position.
