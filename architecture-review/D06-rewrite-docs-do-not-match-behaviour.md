# D06 — Request-rewriting docs promise host-scheme upgrades and rewrite-then-mock chaining; neither happens

**Severity:** Medium
**Area:** Documentation vs implementation — request rewriting

---

## 1. What is wrong

### (a) Host rewrites do not change the scheme, but the docs show `https://`

`docs/Request-Rewriting.md`, "Host Rewriting":

```yaml
rewrites:
  - from: /auth/{endpoint}
    to: /v1/{endpoint}
    host: auth-service.example.com
```

> **Request flow:**
> - `GET /auth/login` → `GET https://auth-service.example.com/v1/login`
> - `POST /payment/process` → `POST https://payment-service.example.com/v2/process`

The `host` value in the example carries no scheme. Following the code:

1. The rewrite middleware stores `m.rewrite.Host.HostPort()` — hostname and port
   only, no scheme — in the request context
   ([`internal/handler/rewrite/handler.go:36`](../internal/handler/rewrite/handler.go#L36)).
2. The proxy builds ad-hoc replacers from it
   ([`internal/handler/proxy/handler.go:132`](../internal/handler/proxy/handler.go#L132)):
   `urlreplacer.NewReplacer(req.URL.Host, rewriteHost)`.
3. `NewReplacer` derives the target scheme with `extractScheme(target)`
   ([`internal/urlreplacer/replacer.go:70`](../internal/urlreplacer/replacer.go#L70)),
   which matches `^(https?):` — no match for a bare hostname, so `scheme` is `""`
   and **no scheme hook is installed**
   ([`internal/urlreplacer/replacer.go:74`](../internal/urlreplacer/replacer.go#L74)).
4. `Replace` therefore substitutes `${scheme}` with the **incoming** request's
   scheme ([`internal/urlreplacer/replacer.go:85`](../internal/urlreplacer/replacer.go#L85)).

Since the example's mapping is `from: http://localhost`, the rewritten request
goes to `http://auth-service.example.com/v1/login`, not `https://`. Writing
`host: https://auth-service.example.com` would produce the documented result —
but the docs' own `host` property table says only "Override upstream host for this
rewrite rule", never mentioning that a scheme may or should be included.

There is a second consequence: `Replacer.IsTargetSecure()` returns `false` when
no scheme was given ([`internal/urlreplacer/replacer.go:114`](../internal/urlreplacer/replacer.go#L114)),
so cookies forwarded on a rewritten request get `Secure = false`
([`internal/handler/proxy/helpers.go:44`](../internal/handler/proxy/helpers.go#L44)).

### (b) "Combining Rewrites with Other Features" cannot work

```yaml
rewrites:
  - from: /old-api/{resource}
    to: /v2/api/{resource}
mocks:
  - path: /v2/api/health
    response: { code: 200, raw: '{"status": "healthy"}' }
cache:
  - /v2/api/users/**
```

A rewrite route's handler is `Mddleware(rewriteMiddleware, defaultHandler)`
([`internal/handler/router/router.go:85`](../internal/handler/router/router.go#L85)),
where `defaultHandler` is the HAR→cache→OPTIONS→**proxy** chain. The rewrite
middleware mutates `request.URL` and calls `next`
([`internal/handler/rewrite/handler.go:23`](../internal/handler/rewrite/handler.go#L23)) —
`next` being that chain, **not the router**. So `GET /old-api/health` is rewritten
to `/v2/api/health` and sent straight upstream; the mock at that path is never
consulted.

The `cache:` part of the example *does* work, because the cache middleware sits
inside the same chain and sees the rewritten URL. So the example is half-true,
which is harder to debug than wholly false.

### (c) "supports wildcards" is imprecise

The property table says `from` and `to` support "wildcards". They support
**gorilla/mux path variables** (`{name}`), which have specific semantics: a
variable matches one path segment and does not cross `/`. `docs/Configuration.md`
separately (and correctly) uses "named placeholders" for the analogous host
feature. The word "wildcard" also appears in the docs for globs
(`docs/Response-Caching.md`) where it means something different again
(doublestar). Three different pattern languages, one word.

Note the substitution is a plain `strings.ReplaceAll` over `{name}` tokens
([`internal/handler/rewrite/handler.go:44`](../internal/handler/rewrite/handler.go#L44)),
so a `to` value containing `{name}` for a variable that does not exist in `from`
is left literally in the outgoing path rather than erroring.

## 2. Why it matters

Rewriting is the feature users reach for when they need something non-obvious, so
the documentation carries more weight than usual. Both defects fail silently: the
request goes somewhere, just not where the docs said. Debugging (b) is especially
unpleasant because the mock exists, the path matches, and it still never fires.

## 3. Recommended fix

**For (a) — decide and document, then implement:**

- Simplest correct fix: **document that `host` may carry a scheme**
  (`host: https://auth-service.example.com`), fix the two example flows, and note
  that without a scheme the incoming request's scheme is preserved.
- Better: parse `host` as a `Host` with an optional scheme (it already is a
  `urlt.Host` — [`internal/config/rewrite.go:12`](../internal/config/rewrite.go#L12)), and
  **carry the scheme through the context** instead of only `HostPort()`. Then
  `host: https://…` and `host: //…` both behave predictably, and the cookie
  `Secure` flag follows.

**For (b) — make the docs match the code, or the code match the docs:**

- If rewrites are meant to be a pre-proxy path transform only (the current
  behaviour), **delete the "Combining Rewrites with Other Features" mocks example**
  and add an explicit note: *"Rewritten requests are sent upstream directly; they
  are not re-matched against mocks, scripts or static routes."*
- If chaining is intended (which is what the example implies and what users will
  expect), implement re-dispatch into the mapping subrouter with a depth guard —
  see [A06](A06-cross-cutting-middleware-attached-to-one-branch-only.md).

**For (c):** standardise the vocabulary across the docs — "path variables" for
mux `{name}`, "host placeholders" for mapping `{name}`, "glob patterns" for cache
`**`. Add a one-line definition of each to `docs/Home.md`'s terminology table
(which currently defines "Rewrite" but not the pattern syntax).

Also add a validation error for `to` referencing a variable not present in
`from`, so the `strings.ReplaceAll` no-op becomes a config error.

## 4. Why this is better

- Users get requests where they expect them, and cookies with the right `Secure`
  flag.
- Whichever choice is made for (b), it becomes a stated contract instead of an
  accident of where the middleware was attached.
- Consistent pattern vocabulary removes a real source of confusion: today a reader
  who learns `{repo}` in host mappings reasonably assumes `*` works in rewrites (it
  does not) or that `{id}` works in cache globs (it does not).

## 5. Trade-offs and migration considerations

- **Carrying the scheme through the rewrite context is a small behaviour change**
  for anyone who wrote `host: https://…` today: currently the scheme is parsed into
  `urlt.Host.Scheme` but discarded by `HostPort()`, so such users are *already*
  getting the wrong scheme and would be fixed by the change.
- **Implementing rewrite re-dispatch is the larger option** and needs a loop guard
  and a decision about whether HAR/cache see the pre- or post-rewrite URL. Do not
  attempt it independently of [A06](A06-cross-cutting-middleware-attached-to-one-branch-only.md).
- **The per-request replacer construction in `createReplacers`**
  ([`internal/handler/proxy/handler.go:132`](../internal/handler/proxy/handler.go#L132))
  compiles two regexps on every rewritten request; if the rewrite path is touched,
  hoist those to config time.
- Existing integration coverage is in
  [`tests/integration/rewrite/rewrite_test.go`](../tests/integration/rewrite/rewrite_test.go);
  extend it with a host-rewrite scheme assertion and a "rewrite does not reach a
  mock" (or "does reach") assertion so the chosen contract is pinned.

## 6. Code and document references

| What | Where |
| --- | --- |
| Rewrite stores host without scheme | [`internal/handler/rewrite/handler.go:36`](../internal/handler/rewrite/handler.go#L36) |
| Rewrite calls the proxy chain, not the router | [`internal/handler/rewrite/handler.go:23`](../internal/handler/rewrite/handler.go#L23), [`internal/handler/router/router.go:85`](../internal/handler/router/router.go#L85) |
| Variable substitution by `ReplaceAll` | [`internal/handler/rewrite/handler.go:44`](../internal/handler/rewrite/handler.go#L44) |
| Ad-hoc replacers built per request | [`internal/handler/proxy/handler.go:132`](../internal/handler/proxy/handler.go#L132) |
| Scheme derived from target only | [`internal/urlreplacer/replacer.go:70`](../internal/urlreplacer/replacer.go#L70), [`:85`](../internal/urlreplacer/replacer.go#L85) |
| `IsTargetSecure` → cookie `Secure` | [`internal/urlreplacer/replacer.go:114`](../internal/urlreplacer/replacer.go#L114), [`internal/handler/proxy/helpers.go:44`](../internal/handler/proxy/helpers.go#L44) |
| `host` config type | [`internal/config/rewrite.go:12`](../internal/config/rewrite.go#L12) |
| Claims | [`docs/Request-Rewriting.md`](../docs/Request-Rewriting.md) ("Host Rewriting", "Combining Rewrites with Other Features", property table) |
