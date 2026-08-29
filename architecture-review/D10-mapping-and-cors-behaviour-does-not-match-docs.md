# D10 — Assorted config and CORS claims that the implementation contradicts

**Severity:** Medium (each item individually small; collectively they make the reference untrustworthy)
**Area:** Documentation vs implementation — mappings, CORS, proxy

---

This report collects the remaining verified discrepancies that do not warrant a
dedicated document. Each is stated with its evidence and its fix.

---

## Finding 1 — The documented `proxy:` example is rejected at startup

`docs/Configuration.md:106` shows:

```yaml
proxy: localhost:8080
```

and `docs/Configuration.md:451` states that proxy values may be given as
"Host and port: `proxy.example.com:8080` (assumes HTTP)".

`ValidateProxy` requires an absolute URL with both scheme and host
([`internal/config/mapping.go:64-77`](../internal/config/mapping.go#L64)). Verified:

```
ValidateProxy("proxy", "localhost:8080")     → proxy is not a valid URL
ValidateProxy("proxy", "http://proxy:8080")  → nil
```

`url.Parse("localhost:8080")` yields `Scheme: "localhost"`, `Opaque: "8080"`,
`Host: ""`, hence the rejection. A config using the documented form fails to
start.

Note the "assumes HTTP" sentence is about *environment variables*
(`HTTP_PROXY=proxy.example.com:8080`), where Go's `http.ProxyFromEnvironment` does
accept the bare form — so the sentence is correct in its own context but sits
directly above a config example that is not.

**Fix:** either accept the bare form in `ValidateProxy` (prepend `http://` when no
scheme is present, matching `ProxyFromEnvironment`'s behaviour — recommended, it
is what users expect) or change the example to `proxy: http://localhost:8080`. Do
not leave the two inconsistent.

---

## Finding 2 — `//` scheme-agnostic mappings do not listen on both schemes

`docs/Configuration.md`, "Protocol Scheme Mapping":

> Using `//` as the scheme creates a mapping that **matches both HTTP and HTTPS
> requests**.
> ```yaml
> mappings:
>   - from: //localhost:8080
>     to: https://site.com
> ```

Verified behaviour:

```
ParseHost("//localhost:8080")          → {Scheme:"" Hostname:"localhost" Port:"8080"}
NormaliseMappings(...)                  → http://localhost:8080
GroupByPort()                           → [{Port:8080 Scheme:http ...}]
```

`normalizeHost` forces an empty scheme to `http`
([`internal/config/helpers.go:78`](../internal/config/helpers.go#L78)), and the port group's
scheme decides whether the listener is TLS-wrapped
([`internal/di/public_api.go:188`](../internal/di/public_api.go#L188)). A single TCP
listener is either TLS or plaintext; it cannot be both. So `//` produces a
plain-HTTP listener.

What *is* scheme-agnostic is the **URL matching regexp**, which always accepts an
optional `(http(s?):)?//` prefix regardless of the configured scheme
([`internal/urlreplacer/helpers.go:89`](../internal/urlreplacer/helpers.go#L89)) — but that
matters only for the rewrite target, not for which port/protocol is served.

The project's own [`.uncors.yaml`](../.uncors.yaml) uses `from: //localhost:3000`,
so this is not a hypothetical.

**Fix:** document `//` accurately — it means "no explicit scheme; defaults to
HTTP for the listener, and preserves the request's scheme when rewriting the
target". If genuine dual-scheme serving is wanted, it requires two listeners
(one plain, one TLS) for the same mapping, which is a feature, not a doc fix.
`to: //site.com` (preserve the source scheme upstream) *does* work as documented
and should be kept.

---

## Finding 3 — The documented default CORS headers do not match what is sent

`docs/Configuration.md`, "OPTIONS Request Handling":

> The default response includes:
> - `Access-Control-Allow-Origin: *`
> - `Access-Control-Allow-Methods: GET, POST, PUT, DELETE, PATCH, OPTIONS`

`ARCHITECTURE.md`, "Request Flow":

> ```
> Access-Control-Allow-Methods: *
> ```

The actual value is a fixed list
([`internal/infra/cors.go:10`](../internal/infra/cors.go#L10)):

```
GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, HEAD, LINK, OPTIONS
```

(note `HEAD` appears twice). Three documents, three different values, none
correct.

Also undocumented: for **preflight** requests the middleware echoes the request's
`Origin`, `Access-Control-Request-Headers` and `Access-Control-Request-Method`
back rather than using the defaults
([`internal/infra/cors.go:44-56`](../internal/infra/cors.go#L44)), and always sets
`Access-Control-Allow-Credentials: true`,
`Access-Control-Max-Age: 86400` and `Access-Control-Expose-Headers: *`. The
credentials header combined with a wildcard origin is rejected by browsers, which
is why the echo behaviour exists — this is worth documenting because it explains
observed behaviour that otherwise looks inconsistent.

**Fix:** document the real header set in one place (`docs/Configuration.md`),
have `ARCHITECTURE.md` link to it, and deduplicate `HEAD` in the constant.

---

## Finding 4 — Host placeholders match dots, contrary to the documented rule

`docs/Configuration.md`, "Named Placeholder Mapping":

> A placeholder is written as `{name}` and matches any sequence of characters in
> that hostname segment (**excluding `.` and `/`**).

The generated pattern is `(?P<key>.+)`
([`internal/urlreplacer/helpers.go:103`](../internal/urlreplacer/helpers.go#L103)) — greedy
and dot-inclusive. So `http://{repo}.local.com` matches
`http://a.b.c.local.com` with `repo = "a.b.c"`, and in a multi-placeholder
pattern such as `http://{env}.{service}.local.com` the greedy first group changes
which text each name captures.

**Fix:** either emit `[^.]+` per placeholder (matching the documented and more
useful semantics) or correct the documentation. Emitting `[^.]+` is recommended —
it is what the multi-placeholder examples in the docs assume — and should be
paired with the pattern-compiler work in
[A17](A17-pkg-urlt-is-a-fork-of-net-url.md).

---

## Finding 5 — POST/PUT caching is documented but keys ignore the request body

`docs/Response-Caching.md` documents `methods: [GET, POST, PUT]` and gives a
worked example caching `/api/search` and `/api/query/**`.

The cache key is built from method, hostname, path and query only
([`internal/handler/cache/middleware.go:133`](../internal/handler/cache/middleware.go#L133)).
Two POSTs to the same URL with different bodies collide, and the second receives
the first's response. See [A16](A16-cache-design-issues.md).

**Fix:** include a body hash in the key for body-bearing methods, or remove
POST/PUT from the documented examples and state that only body-less methods can
be cached.

---

## Finding 6 — Smaller inaccuracies

| Claim | Location | Reality |
| --- | --- | --- |
| "No mappings configured" error | `docs/Troubleshooting.md` | The actual message is `mappings must not be empty` ([`internal/config/config.go:101`](../internal/config/config.go#L101)) |
| Version "0.6.1" in the wiki header | [`docs/Home.md`](../docs/Home.md) | Hand-maintained; will drift from releases. Generate it or drop it. |
| `cache-config` marked as having no default | `docs/Configuration.md` Global Properties table | It has real defaults: 30 m TTL, 100 MB, `[GET]` ([`internal/config/default.go:15`](../internal/config/default.go#L15)) |
| `docs/Response-Caching.md` Cache Lifecycle: "Evicted … when `max-size` is reached" | | Accurate, but omits that eviction is cost-based (ristretto TinyLFU) and that entries are dropped when a **single** response exceeds the budget |
| `index` described as "Fallback file when requested file not found" | `docs/Static-File-Serving.md` property table | Also used when the resolved path is a **directory** ([`internal/handler/static/middleware.go:89`](../internal/handler/static/middleware.go#L89)) — worth stating |

---

## Why these matter collectively

Individually each is minor. Together with [D01](D01-debug-flag-does-not-exist.md),
[D02](D02-durations-with-spaces-do-not-parse.md), [D03](D03-tilde-paths-are-not-expanded.md)
and [D04](D04-spa-with-mocks-example-is-broken.md), they mean a user cannot rely on
`docs/Configuration.md` — the single most important reference page — without
testing each claim. Several of the errors (the `proxy:` example, the duration
format, the `~` paths) cause **startup failures**, so the first experience of
following the documentation is an error message.

The common cause is that no example in `docs/` is executed by anything. The
highest-leverage fix in this whole documentation set is a test that extracts every
YAML block from `docs/*.md` and runs it through `LoadConfiguration` — see
[D02](D02-durations-with-spaces-do-not-parse.md) §5. That single test would catch
Findings 1 and 2 here, plus D01, D02 and D03, and would keep catching them.

## Trade-offs and migration considerations

- **Accepting bare `host:port` proxy values** (Finding 1) is additive and safe.
- **Changing placeholder matching to `[^.]+`** (Finding 4) is a behaviour change:
  a user relying on `{repo}` capturing a multi-level subdomain would break. It is
  the documented semantics, so treat it as a bug fix, but note it in
  `docs/Migration-Guide.md`.
- **Deduplicating `HEAD`** in the methods constant is cosmetic but visible in every
  response header.
- **Dual-scheme listeners** (Finding 2) are a feature request, not a fix; the doc
  correction should land now regardless.

## Code and document references

| What | Where |
| --- | --- |
| Proxy validation | [`internal/config/mapping.go:64`](../internal/config/mapping.go#L64) |
| Scheme normalisation | [`internal/config/helpers.go:78`](../internal/config/helpers.go#L78) |
| TLS decided by group scheme | [`internal/di/public_api.go:188`](../internal/di/public_api.go#L188) |
| CORS constants and echo behaviour | [`internal/infra/cors.go:10`](../internal/infra/cors.go#L10), [`:44`](../internal/infra/cors.go#L44) |
| Placeholder regexp | [`internal/urlreplacer/helpers.go:103`](../internal/urlreplacer/helpers.go#L103) |
| Scheme-agnostic match prefix | [`internal/urlreplacer/helpers.go:89`](../internal/urlreplacer/helpers.go#L89) |
| Cache key | [`internal/handler/cache/middleware.go:133`](../internal/handler/cache/middleware.go#L133) |
| Cache defaults | [`internal/config/default.go:15`](../internal/config/default.go#L15) |
| Empty-mappings message | [`internal/config/config.go:101`](../internal/config/config.go#L101) |
| Static index-on-directory | [`internal/handler/static/middleware.go:89`](../internal/handler/static/middleware.go#L89) |
| Project's own `//` mapping | [`.uncors.yaml`](../.uncors.yaml) |
