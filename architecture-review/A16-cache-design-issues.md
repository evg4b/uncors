# A16 — The response cache blocks the request goroutine on every write and keys entries on the URL alone

**Severity:** Medium-High
**Area:** Caching

---

## 1. What is wrong with the current approach

### (a) `Set` blocks on `ristretto.Wait()` in the request path

```go
// internal/handler/cache/cache.go:52-56
func (cs *RistrettoCache) Set(key string, value contracts.CachedResponse) {
	cs.storage.SetWithTTL(key, value, CalcCost(&value), cs.ttl)
	cs.storage.Wait()          // ← blocks until ristretto's buffers are processed
}
```

Ristretto is explicitly designed around asynchronous, batched admission — `Set`
enqueues into a ring buffer and returns immediately. `Wait()` blocks the caller
until the buffers have drained. Calling it inside `Set` converts an async cache
into a synchronous one and adds that latency to **every cacheable response**,
right after the body has been written to the client
([`internal/handler/cache/middleware.go:69`](../internal/handler/cache/middleware.go#L69)).
The one property that makes ristretto worth the dependency is deliberately
disabled.

`Wait()` is presumably there to make tests deterministic (`RistrettoCache.Wait()`
also exists as a separate public method at
[`internal/handler/cache/cache.go:57`](../internal/handler/cache/cache.go#L57), and is
never called outside tests). Test determinism should not be paid for on the hot
path.

### (b) The cache key ignores everything except method, host, path and query

```go
// internal/handler/cache/middleware.go:141-152
return fmt.Sprintf("[%s]%s%s?%s", method, urlt.URL_Hostname(url), url.Path, strings.Join(items, ";"))
```

Notably absent:

- **The request body.** `docs/Response-Caching.md` documents
  `methods: [GET, POST, PUT]` and gives a worked example caching `/api/search` and
  `/api/query/**` with POST. Two POSTs to `/api/search` with different bodies
  produce the same key, so **the second request receives the first one's
  response**. This is a silent correctness bug in a documented feature.
- **`Vary`-relevant request headers.** An upstream that varies on
  `Accept-Language`, `Accept-Encoding` or `Authorization` will have one variant
  served to all clients. The stored response's `Vary` header is available in
  `capture.Header` and is ignored.
- **The port and scheme.** `urlt.URL_Hostname(url)` strips the port, so
  `http://api.local:3000/x` and `http://api.local:4000/x` — two different mappings
  in the same process, sharing one global cache instance — collide.

### (c) One global cache instance shared by all mappings

`Container.Cache` is memoised for the process
([`internal/di/public_api.go:110`](../internal/di/public_api.go#L110)), so all mappings
share one keyspace and one size budget. Combined with (b)'s missing port, mappings
that proxy different upstreams under the same hostname on different ports share
entries. There is also no way to give one mapping a shorter TTL.

### (d) `cache-config` changes never take effect on reload

Because of the `sync.Once` in `factory1`
([`internal/di/factory.go:31`](../internal/di/factory.go#L31)) — see
[A02](A02-di-container-leaks-resources-on-every-config-reload.md) — editing
`expiration-time`, `max-size` or `methods` and saving does nothing until restart,
even though the UI prints "Server restarted". Stale entries also survive a reload
that removes a `cache:` glob.

### (e) HTTP cache semantics are ignored entirely

Only the status code is consulted (`Is2xxCode`,
[`internal/handler/cache/middleware.go:72`](../internal/handler/cache/middleware.go#L72)).
`Cache-Control: no-store` / `private` / `max-age`, `Set-Cookie` on a cached
response, and conditional requests (`If-None-Match`, `If-Modified-Since`) are all
disregarded. Caching a response that carries `Set-Cookie` and replaying it to a
different client is the most dangerous of these.

### (f) `NumCounters` is a fixed `1e5` regardless of `MaxSize`

[`internal/handler/cache/cache.go:22`](../internal/handler/cache/cache.go#L22). Ristretto's
guidance is roughly 10× the expected number of items; with a 100 MB default budget
and small JSON responses the item count can far exceed 10 000, degrading the
admission policy's accuracy.

### (g) A panic on write failure

```go
// internal/handler/cache/middleware.go:105
if _, err := writer.Write(cachedResponse.Body); err != nil { panic(err) }
```

A client that disconnects mid-response panics the handler goroutine.

## 2. Why it is an architectural problem

- **The cache is presented as an HTTP response cache but implements a URL→bytes
  map.** The gap between those two things is exactly where the correctness bugs
  live (POST bodies, `Vary`, `Set-Cookie`, `no-store`). Users reasonably assume
  HTTP semantics because the feature is described in HTTP terms.
- **Cache identity, lifetime and configuration are all process-global**, while the
  thing being cached is per-mapping. That mismatch is what produces the
  cross-mapping collisions and the non-reloadable config.
- **A deliberate architectural choice (async admission) is negated by one line**,
  which suggests the caching layer's performance model was never load-tested.

## 3. What the recommended approach is instead

**1. Remove `Wait()` from `Set`.** Keep the public `Wait()` method and call it
from tests only:

```go
func (cs *RistrettoCache) Set(k string, v contracts.CachedResponse) {
	cs.storage.SetWithTTL(k, v, CalcCost(&v), cs.ttl)
}
// tests: cache.Set(...); cache.Wait(); assert...
```

**2. Make the key complete and explicit.**

```go
type cacheKey struct {
	Mapping string   // stable mapping identifier, not just hostname
	Method  string
	Host    string   // host:port
	Path    string
	Query   string   // canonicalised
	Vary    string   // hash of the request headers named by the stored Vary
	Body    string   // hash of the body, only for methods with a body
}
```

Hash it into a string once. For body-bearing methods, read and rehash the body via
the same buffered-body mechanism HAR uses (and cap it —
[A07](A07-response-writer-drops-streaming-capabilities.md)); if the body exceeds the
cap, do not cache.

Note that `Vary` requires two lookups (fetch the stored variant list, then key on
it) or storing a small variant index per URL. If that complexity is unwanted, the
defensible alternative is to **refuse to cache responses that carry `Vary`** —
but that decision must be made and documented, not left implicit.

**3. Respect the minimum viable HTTP semantics.** Do not store a response when:
`Cache-Control` contains `no-store` or `private`; the response carries
`Set-Cookie`; the request carries `Authorization` (unless `public` is present).
This is a short function and eliminates the dangerous cases.

**4. Move the cache into the per-generation scope** so `cache-config` reloads,
and give it a per-mapping namespace prefix in the key (or a per-mapping instance
if per-mapping TTLs are wanted later).

**5. Derive `NumCounters` from `MaxSize`** (e.g. `maxSize / expectedEntrySize * 10`),
or expose it.

**6. Replace the `panic(err)` with a logged write failure.**

## 4. Why the proposed approach is better

- **Removing `Wait()` returns the cache to O(1) non-blocking writes**, which is the
  entire reason ristretto was chosen. This is a one-line, zero-risk latency win on
  every cached response.
- **Correct keys eliminate a class of "why did I get the wrong response?" bugs**
  that are extremely hard for a user to diagnose — they look like upstream bugs.
- **Documented POST caching starts working correctly** instead of silently
  returning the wrong body.
- **Refusing to cache `Set-Cookie`/`private` responses** removes a genuine
  cross-session data-leak risk when several browser profiles or test users share
  one uncors instance.
- **A reloadable cache config** matches what the UI already claims happened.

## 5. Trade-offs and migration considerations

- **Body hashing costs a read and a hash per cacheable request with a body.** Only
  do it for methods actually listed in `cache-config.methods`; for the default
  (`GET` only) the cost is zero.
- **Adding `Vary`/`Set-Cookie` rules will reduce hit rates** for some upstreams.
  That is the correct trade, but call it out in the docs so users understand why a
  previously-cached endpoint stopped caching.
- **Changing the key format invalidates the in-memory cache**, which is harmless
  (it is not persisted).
- **Removing `Wait()` from `Set` will make existing cache tests flaky** unless they
  are updated to call `Wait()` explicitly —
  [`internal/handler/cache/middleware_test.go`](../internal/handler/cache/middleware_test.go)
  and [`testing/testutils/cache.go`](../testing/testutils/cache.go) are the places to
  look.
- Per-mapping cache instances would be cleaner but multiply the memory budget;
  a shared instance with namespaced keys is the pragmatic middle ground.

## 6. Code references

| What | Where |
| --- | --- |
| Blocking `Wait()` in `Set` | [`internal/handler/cache/cache.go:52`](../internal/handler/cache/cache.go#L52) |
| Unused public `Wait()` | [`internal/handler/cache/cache.go:57`](../internal/handler/cache/cache.go#L57) |
| Cache key construction | [`internal/handler/cache/middleware.go:133`](../internal/handler/cache/middleware.go#L133) |
| Store decision (status code only) | [`internal/handler/cache/middleware.go:72`](../internal/handler/cache/middleware.go#L72) |
| Panic on write | [`internal/handler/cache/middleware.go:109`](../internal/handler/cache/middleware.go#L109) |
| Global memoised instance | [`internal/di/public_api.go:110`](../internal/di/public_api.go#L110), [`internal/di/factories.go:37`](../internal/di/factories.go#L37) |
| `sync.Once` ignoring config | [`internal/di/factory.go:31`](../internal/di/factory.go#L31) |
| Fixed `NumCounters` | [`internal/handler/cache/cache.go:22`](../internal/handler/cache/cache.go#L22) |
| Documented POST caching | [`docs/Response-Caching.md`](../docs/Response-Caching.md) |
