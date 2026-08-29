# D04 — The documented "SPA with API Proxying" configuration does not work: the static handler shadows every mock

**Severity:** High (the flagship example of a flagship feature is broken)
**Area:** Documentation vs implementation — routing
**Status:** Reproduced

---

## 1. What is wrong

`docs/Static-File-Serving.md` presents this as the canonical SPA setup:

```yaml
mappings:
  - from: http://localhost:3000
    to: https://api.example.com
    statics:
      - path: /
        dir: ~/my-app/build
        index: index.html
    mocks:
      - path: /api/health
        response:
          code: 200
          raw: '{"status": "ok"}'
```

with the explicit claim:

> In this configuration:
> - SPA files are served from the root path `/`
> - The `/api/health` endpoint is mocked
> - All other `/api/*` requests are proxied to `https://api.example.com`

None of the last two bullets is true. `registerMapping` registers static routes
**before** mock routes ([`internal/handler/router/router.go:66`](../internal/handler/router/router.go#L66)
vs [`:70`](../internal/handler/router/router.go#L70)), and `registerPrefixHandler`
registers a `PathPrefix("/")` route
([`internal/handler/router/router_helpres.go:49`](../internal/handler/router/router_helpres.go#L49)).
`gorilla/mux` matches routes in registration order and returns the first match, so
`PathPrefix("/")` claims **every** path in the mapping. Because `index` is set,
the static middleware falls back to `index.html` for any path that is not a real
file ([`internal/handler/static/middleware.go:107`](../internal/handler/static/middleware.go#L107))
and never calls `next`.

Reproduced with the project's own router test harness:

```go
Statics: {Dir: "/dist", Path: "/", Index: "index.html"}
Mocks:   {Path: "/api/health", Response: {Code: 200, Raw: "MOCK-HEALTH"}}
```
```
GET /api/health -> 200 "SPA-INDEX"
```

The mock is unreachable, and so is the proxy: with `index` set, *nothing* under
that mapping ever reaches the upstream. A user following this documentation gets
an SPA that serves `index.html` for its own API calls — which manifests as
"my API returns HTML" and is very hard to attribute to route ordering.

The same page's "Request handling behavior" section states the general rule:

> If a file does not exist and `index` is not set, the request is forwarded
> upstream (proxy mode)

That part is accurate (the static middleware returns `errNotHandled` and calls
`next` — [`internal/handler/static/middleware.go:38`](../internal/handler/static/middleware.go#L38)),
but "Proxy Mode" then says the request "passes to the next handler (**mock** or
upstream)", which is also wrong: `next` is the proxy chain, not the router, so a
static miss can never fall through to a mock
([A06](A06-cross-cutting-middleware-attached-to-one-branch-only.md)).

`docs/Real-World-Examples.md` and `docs/HAR-Collector.md` both contain
configurations with the same shape (root or broad static + mocks), so the problem
is not confined to one page.

## 2. Why it is an architectural problem, not just a doc bug

Route precedence is **implicit in the order of `for` loops** inside
`registerMapping`. It is:

1. not documented anywhere,
2. not expressible or overridable by the user,
3. not consistent with the two-pass specificity logic the code *does* apply within
   mocks and scripts ([`internal/handler/router/routes.go:5`](../internal/handler/router/routes.go#L5)),
   which sorts specific matchers ahead of path-only ones — the very principle that
   should also place a `PathPrefix("/")` catch-all last.

So the code already knows that specificity should determine order; it just does
not apply that knowledge across route kinds.

## 3. Recommended fix

**Primary — change the precedence (code).** Register terminal routes in
specificity order within each mapping:

1. mocks and scripts with non-path matchers (method/query/header)
2. mocks and scripts with path-only matchers
3. rewrites
4. statics, with `path: /` catch-alls registered last of all
5. the mapping default (proxy)

This makes the documented example behave as documented and is consistent with the
existing intra-kind logic. It is part of the wider router restructuring in
[A06](A06-cross-cutting-middleware-attached-to-one-branch-only.md).

**Secondary — warn on shadowing.** At config load, detect when a static `path`
prefix (with `index` set) covers a configured mock or script path, and emit:

```
WARNING  static mount "/" with index "index.html" shadows mock "/api/health";
         the mock will never be reached. Mount the static under a narrower path,
         or remove the index fallback.
```

This is valuable even after the precedence change, because it catches the
genuinely ambiguous cases.

**Also — correct the docs regardless.** Add a "Route precedence" section to
`docs/Configuration.md` stating the order explicitly, and fix
`docs/Static-File-Serving.md`'s "Proxy Mode" wording ("passes to the next handler
(**upstream**)" — not mock).

## 4. Why this is better

- **The single most common uncors configuration starts working.** SPA + mocked
  endpoints + proxied API is the archetypal use case; it is what the feature set
  exists for.
- **Precedence becomes a documented contract** rather than an emergent property of
  loop order, so future route kinds have a rule to follow.
- **The warning turns a silent, mystifying failure into a startup message** that
  names both the static mount and the shadowed mock.
- It aligns the cross-kind behaviour with the intra-kind behaviour the code
  already implements deliberately.

## 5. Trade-offs and migration considerations

- **Reordering routes is a behaviour change.** A user who currently relies on a
  static file shadowing a mock (unlikely, but possible — e.g. a `/api/health.json`
  file that happens to win) would see different behaviour. Document it in
  `docs/Migration-Guide.md`.
- **`PathPrefix` catch-alls interact with the `Path`+redirect pair** that
  `registerPrefixHandler` installs
  ([`internal/handler/router/router_helpres.go:44-50`](../internal/handler/router/router_helpres.go#L45)):
  the bare `/` route issues a 307 redirect to `/`, which for `path: /` is a
  self-redirect. Worth checking during the change — `normalizePath("/")` returns
  `("", "/")`, so the redirect route is `Path("")`, which mux will not match; the
  behaviour happens to be benign today but is accidental.
- **The shadowing check needs to understand mux path variables** (`/users/{id}`)
  to avoid false positives; a conservative version that only flags literal-prefix
  containment is enough to catch the documented case.
- Add an integration test in `tests/integration/static` or
  `tests/integration/routing` asserting that a mock beats a root static mount —
  this exact scenario currently has no coverage.

## 6. Code and document references

| What | Where |
| --- | --- |
| Statics registered before mocks | [`internal/handler/router/router.go:66`](../internal/handler/router/router.go#L66), [`:70`](../internal/handler/router/router.go#L70) |
| `PathPrefix` catch-all registration | [`internal/handler/router/router_helpres.go:45`](../internal/handler/router/router_helpres.go#L45) |
| Index fallback prevents `next` | [`internal/handler/static/middleware.go:107`](../internal/handler/static/middleware.go#L107), [`:38`](../internal/handler/static/middleware.go#L38) |
| Static's `next` is the proxy, not the router | [`internal/handler/router/router.go:68`](../internal/handler/router/router.go#L68) |
| Existing intra-kind specificity pass | [`internal/handler/router/routes.go:5`](../internal/handler/router/routes.go#L5) |
| Broken example | [`docs/Static-File-Serving.md`](../docs/Static-File-Serving.md) ("SPA with API Proxying") |
| "mock or upstream" claim | [`docs/Static-File-Serving.md`](../docs/Static-File-Serving.md) ("Proxy Mode") |
| Same shape elsewhere | [`docs/HAR-Collector.md`](../docs/HAR-Collector.md) ("Combine with Other Features"), [`docs/Real-World-Examples.md`](../docs/Real-World-Examples.md) |
