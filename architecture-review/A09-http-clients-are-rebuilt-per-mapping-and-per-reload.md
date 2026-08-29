# A09 — A fresh `http.Transport` is created per port group and per reload, and none are ever closed

**Severity:** Medium-High (performance + resource leak)
**Area:** HTTP client lifecycle

---

## 1. What is wrong with the current approach

`infra.MakeHTTPClient` allocates a brand-new `http.Transport` on every call:

```go
// internal/infra/httpclient.go:12-33
func MakeHTTPClient(proxy string) *http.Client {
	transport := &http.Transport{Proxy: http.ProxyFromEnvironment}
	if proxy != "" {
		parsedURL, err := url.Parse(proxy)
		if err != nil { panic(...) }
		transport.Proxy = http.ProxyURL(parsedURL)
	}
	return &http.Client{CheckRedirect: ..., Transport: transport, Timeout: defaultTimeout}
}
```

It is called from two places, both of which run repeatedly:

- `Container.ProxyHandler` — [`internal/di/public_api.go:152`](../internal/di/public_api.go#L152) —
  invoked once **per port group** from `Targets`
  ([`internal/di/public_api.go:172`](../internal/di/public_api.go#L172)), and again for
  every port group on **every config reload**.
- `Container.VersionChecker` — [`internal/di/public_api.go:95`](../internal/di/public_api.go#L95).

So a config with three distinct `from` ports creates three independent connection
pools to the same upstreams. Save the config file five times and there are
eighteen. None of them is ever `CloseIdleConnections()`d, because the proxy
handler is discarded without a teardown step
([A02](A02-di-container-leaks-resources-on-every-config-reload.md)) — the idle
keep-alive connections from every previous generation stay open until the
upstream or the OS closes them.

Further issues with the transport configuration itself:

- **Every tunable is left at the zero value.** `MaxIdleConnsPerHost` therefore
  defaults to 2 (`http.DefaultMaxIdleConnsPerHost`). For a dev proxy in front of a
  browser issuing 6+ parallel requests to one host, that means connections are
  constantly being created and torn down — the single easiest performance win
  available in this codebase and it is left on the table. `http.DefaultTransport`
  by contrast sets `MaxIdleConns: 100`, `IdleConnTimeout: 90s`,
  `TLSHandshakeTimeout: 10s`, `ExpectContinueTimeout: 1s` and enables HTTP/2 —
  none of which this transport has. Notably, **HTTP/2 to upstreams is disabled**,
  because `ForceAttemptHTTP2` is false and a manually-constructed `Transport`
  does not auto-upgrade.
- **`Timeout: 5 * time.Minute` is a whole-request timeout** including body
  transfer ([`internal/infra/httpclient.go:10`](../internal/infra/httpclient.go#L10)).
  That caps long downloads and makes streaming responses impossible even if
  [A07](A07-response-writer-drops-streaming-capabilities.md) were fixed. The right
  knobs are `DialContext.Timeout`, `TLSHandshakeTimeout` and
  `ResponseHeaderTimeout`, not a global deadline.
- **It panics on a bad proxy URL** rather than returning an error, from inside a
  factory called during router construction. Config validation does check the
  proxy (`ValidateProxy`, [`internal/config/mapping.go:64`](../internal/config/mapping.go#L64)),
  so the panic is currently unreachable — which makes it dead defensive code that
  will become reachable the moment validation changes.

## 2. Why it is an architectural problem

- **Connection pooling is a process-level resource, modelled as a per-handler
  value.** The whole point of `http.Transport` is that it is shared and long-lived;
  the Go docs say so explicitly. Constructing one per handler defeats it.
- **The lifetime mismatch produces a leak that is invisible in testing**, because
  test suites don't reload configs dozens of times and don't keep upstream
  keep-alives.
- **Upstream transport policy is not configurable at all.** There is no way for a
  user to say "my upstream is slow, raise the response-header timeout" or "don't
  reuse connections". For a tool whose job is talking to arbitrary upstreams, the
  transport is a first-class configuration surface that currently does not exist.

## 3. What the recommended approach is instead

**One transport per distinct upstream policy, owned by the composition root.**

```go
// internal/infra
type ClientPool struct {
	mu sync.Mutex
	byProxy map[string]*http.Client
}

func (p *ClientPool) For(proxyURL string) (*http.Client, error)  // memoised
func (p *ClientPool) CloseIdle()                                 // on shutdown / reload
```

with a transport derived from `http.DefaultTransport` rather than built from
zero:

```go
tr := http.DefaultTransport.(*http.Transport).Clone()
tr.MaxIdleConnsPerHost   = 32            // browsers open ~6 per host; leave headroom
tr.ResponseHeaderTimeout = 60 * time.Second
tr.ForceAttemptHTTP2     = true
if proxyURL != "" { tr.Proxy = http.ProxyURL(parsed) } // else ProxyFromEnvironment
```

and a client with **no global `Timeout`** (per-request deadlines come from the
inbound request's context, which the proxy already propagates via
`http.NewRequestWithContext`).

`MakeHTTPClient` should return `(*http.Client, error)` instead of panicking, and
the pool should be closed as part of the generation teardown from
[A02](A02-di-container-leaks-resources-on-every-config-reload.md).

Optionally expose a small `upstream:` config block (`max-idle-conns-per-host`,
`response-header-timeout`, `insecure-skip-verify` for self-signed dev upstreams —
a genuinely common need for a dev proxy that is currently impossible).

## 4. Why the proposed approach is better

- **Measurably faster.** Raising `MaxIdleConnsPerHost` from 2 to 32 eliminates
  most TCP+TLS handshakes for a browser-driven workload; on an HTTPS upstream each
  avoided handshake is a full round trip plus asymmetric crypto. This is the
  single largest architectural performance lever in the project.
- **HTTP/2 to upstreams** is enabled, which matters for the CDN- and
  gateway-hosted APIs uncors is typically pointed at.
- **No per-reload leak**, and idle connections are released deterministically.
- **Streaming and large transfers stop being capped** by an arbitrary 5-minute
  ceiling.
- **Errors are returned, not panicked**, so a future config path that skips
  validation degrades gracefully.

## 5. Trade-offs and migration considerations

- **Removing the global `Timeout` removes a safety net.** Compensate with
  `ResponseHeaderTimeout` (bounds a hung upstream that accepts the connection and
  never replies) plus the inbound request context (bounds a client that gives up).
  Do not simply delete the timeout without adding these.
- **Sharing one transport across mappings shares its connection pool.** That is
  desirable, but it means one misbehaving upstream can consume idle-connection
  budget. `MaxConnsPerHost` can bound this if it becomes an issue.
- **Enabling HTTP/2 changes header casing and trailer behaviour** observed by
  upstreams; extremely unlikely to matter but worth noting for anyone debugging a
  behaviour difference after the change.
- Because `ProxyHandler` currently receives the client by option
  ([`internal/handler/proxy/handler.go:166`](../internal/handler/proxy/handler.go#L166)),
  the migration is a one-line change at the call site plus the new pool type.

## 6. Code references

| What | Where |
| --- | --- |
| Transport built from zero value | [`internal/infra/httpclient.go:12`](../internal/infra/httpclient.go#L12) |
| Global 5-minute timeout | [`internal/infra/httpclient.go:10`](../internal/infra/httpclient.go#L10) |
| Panic on bad proxy URL | [`internal/infra/httpclient.go:20`](../internal/infra/httpclient.go#L20) |
| Called per port group | [`internal/di/public_api.go:152`](../internal/di/public_api.go#L152), [`:172`](../internal/di/public_api.go#L172) |
| Called for version check | [`internal/di/public_api.go:95`](../internal/di/public_api.go#L95) |
| Client injected into proxy handler | [`internal/handler/proxy/handler.go:166`](../internal/handler/proxy/handler.go#L166) |
| Proxy URL validation (makes the panic dead) | [`internal/config/mapping.go:64`](../internal/config/mapping.go#L64) |
