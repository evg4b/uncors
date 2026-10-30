# Integration test coverage

Every documented feature of UNCORS, and whether a test in this folder starts
the proxy, sends a real request and checks both what the upstream received and
what the client got. Features follow the pages in [`docs/`](../../docs).

✅ covered · ❌ not covered

## Summary

| Area                                            |     Covered |         |              |
| ----------------------------------------------- | ----------: | ------: | ------------ |
| [Proxying](#proxying)                           |       12/18 |     67% | `███████░░░` |
| [Hosts and ports](#hosts-and-ports)             |         6/9 |     67% | `███████░░░` |
| [CORS](#cors)                                   |        8/10 |     80% | `████████░░` |
| [OPTIONS handling](#options-handling)           |         4/4 |    100% | `██████████` |
| [Mocks](#mocks)                                 |       11/12 |     92% | `█████████░` |
| [Scripts](#scripts)                             |       12/14 |     86% | `█████████░` |
| [Static files](#static-files)                   |       13/13 |    100% | `██████████` |
| [Rewrites](#rewrites)                           |        8/10 |     80% | `████████░░` |
| [Cache](#cache)                                 |       11/15 |     73% | `███████░░░` |
| [HAR](#har)                                     |        9/11 |     82% | `████████░░` |
| [WebSocket](#websocket)                         |         7/8 |     88% | `█████████░` |
| [Lifecycle and reload](#lifecycle-and-reload)   |         8/9 |     89% | `█████████░` |
| [Validation and errors](#validation-and-errors) |         8/9 |     89% | `█████████░` |
| [CLI and tooling](#cli-and-tooling)             |         1/5 |     20% | `██░░░░░░░░` |
| **Total**                                       | **118/147** | **80%** | `████████░░` |

## Proxying

[Configuration: what UNCORS changes](../../docs/Configuration.md#what-uncors-changes-in-proxied-traffic)

- ✅ Path and query passed unchanged (`//`, `.`, `..`, `%2F`) — `routing/routing_test.go`
- ✅ `Origin` / `Referer` rewritten to the upstream; foreign `Origin` kept — `proxy/proxy_test.go`
- ✅ Body forwarded with its `Content-Length` — `proxy/proxy_test.go`, `har/har_test.go`
- ✅ `POST` / `PUT` / `PATCH` / `DELETE` with body, 2 MB body intact — `proxy/upstream_test.go`
- ✅ Hop-by-hop response headers dropped — `proxy/hop_by_hop_test.go`
- ✅ Hop-by-hop request headers and `Proxy-Authorization` dropped — `proxy/upstream_test.go`
- ✅ `HEAD` keeps the upstream `Content-Length` — `proxy/hop_by_hop_test.go`
- ✅ Event streams delivered chunk by chunk — `proxy/streaming_test.go`
- ✅ `Set-Cookie` domain rewritten, `Secure` on https — `proxy/proxy_test.go`
- ✅ `Location` rewritten to the local host — `proxy/proxy_test.go`
- ✅ Compressed body passed through untouched — `proxy/upstream_test.go`
- ✅ Unreachable upstream → 502 with the reason and CORS — `proxy/upstream_test.go`
- ❌ `304 Not Modified` keeps the upstream `Content-Length`
- ❌ `SameSite=None` becomes `SameSite=Lax` on an http host
- ❌ Slow upstream → 504
- ❌ `https://` upstream
- ❌ Scheme-less `to: //host` uses the incoming scheme
- ❌ Upstream proxy: `proxy`, `--proxy`, `HTTP(S)_PROXY`, `NO_PROXY`

## Hosts and ports

[Configuration: mapping options](../../docs/Configuration.md#mapping-options)

- ✅ `http` and `https` listeners — `proxy/proxy_test.go`, `tls/tls_test.go`
- ✅ Several hosts on one port — `domains/domains_test.go`, `proxy/proxy_test.go`
- ✅ One listener per port — `events/events_test.go`
- ✅ `{name}` placeholder in `from`, path not matched as host — `domains/domains_test.go`
- ✅ Unmapped host → 404 naming the host — `events/events_test.go`
- ✅ TLS: certificate per host, untrusted CA and unknown SNI rejected — `tls/tls_test.go`
- ❌ `{name}` placeholder substituted into `to`
- ❌ Shorthand mapping `http://host: https://target`
- ❌ `--from` / `--to` flags, and replacing a `to` from the file

## CORS

[Configuration: CORS headers](../../docs/Configuration.md#what-uncors-changes-in-proxied-traffic) ·
[Allowed origins](../../docs/Configuration.md#allowed-origins)

- ✅ `Access-Control-Allow-Origin` reflects `Origin`, credentials allowed — `proxy/proxy_test.go`
- ✅ `*` without an `Origin` — `proxy/__snapshots__/proxy_test.snap`
- ✅ `Vary: Origin` added — `cache/cache_test.go`
- ✅ `Date` in `Access-Control-Expose-Headers` — `cors/allowed_origins_test.go`
- ✅ Upstream CORS headers replaced — `cors/allowed_origins_test.go`
- ✅ Same headers on mocks and scripts — `cors/allowed_origins_test.go`
- ✅ No CORS headers on static files — `static/behaviour_test.go`
- ✅ `allowed-origins`: match reflected, other origins get none, preflight 403 — `cors/`
- ❌ Expose list names every response header but `Set-Cookie` and safelisted ones
- ❌ Upstream `Access-Control-Allow-Headers` removed

## OPTIONS handling

[Configuration: OPTIONS request handling](../../docs/Configuration.md#options-request-handling)

- ✅ Default preflight reflects requested method and headers — `options/options_test.go`
- ✅ Custom `code` and `headers` — `options/options_test.go`
- ✅ `disabled: true` forwards to the upstream — `options/options_test.go`
- ✅ Preflight on mocks and scripts; an `OPTIONS` mock wins — `cors/preflight_test.go`

## Mocks

[Response mocking](../../docs/Response-Mocking.md)

- ✅ `raw` body, code, headers — `mock/mock_test.go`
- ✅ `file` body, with 200 and other codes — `mock/mock_test.go`
- ✅ No body (`204`) — `mock/mock_test.go`
- ✅ Headers override the detected `Content-Type` — `mock/matching_test.go`
- ✅ `delay` — `mock/matching_test.go`
- ✅ `method` filter, mismatch proxied — `mock/mock_test.go`
- ✅ `queries` filter — `mock/matching_test.go`
- ✅ `headers` filter — `mock/matching_test.go`
- ✅ Filtered mocks before path-only ones — `mock/matching_test.go`
- ✅ `{id}` path segments — `mock/matching_test.go`
- ✅ File removed after start → 500 with the reason — `mock/matching_test.go`
- ❌ Client disconnect stops the delay

## Scripts

[Script handler](../../docs/Script-Handler.md)

- ✅ Inline script — `script/script_test.go`
- ✅ `file` script, re-read on each request — `script/request_test.go`
- ✅ `request`: method, path, query, host, headers, body — `script/request_test.go`
- ✅ `request.path_params` — `script/script_test.go`
- ✅ Response status, headers, body — `script/request_test.go`
- ✅ `require("json")` — `script/request_test.go`
- ✅ `method` filter, mismatch proxied — `script/request_test.go`
- ✅ Runtime and syntax errors → 500 with the reason — `script/request_test.go`
- ✅ `timeout` — `script/request_test.go`
- ✅ `max-body-size` — `script/request_test.go`
- ✅ `os.exit` refused — `script/request_test.go`
- ✅ Error after the status line aborts the response — `script/request_test.go`
- ❌ `queries` / `headers` filters
- ❌ Client disconnect stops the script

## Static files

[Static file serving](../../docs/Static-File-Serving.md)

- ✅ File served with `Content-Type` from its extension — `static/`
- ✅ `HEAD` — `static/behaviour_test.go`
- ✅ Byte ranges — `static/behaviour_test.go`
- ✅ Conditional request → 304 — `static/behaviour_test.go`
- ✅ `index` fallback for unknown paths (SPA) — `static/static_test.go`
- ✅ Directory with `index` → index — `static/behaviour_test.go`
- ✅ Directory without `index` → upstream — `static/behaviour_test.go`
- ✅ Missing file and path past a file → upstream — `static/static_test.go`
- ✅ `/prefix` redirected to `/prefix/` with the query — `static/static_test.go`
- ✅ Methods other than `GET` / `HEAD` → upstream — `static/behaviour_test.go`
- ✅ `path: /` — `static/static_test.go`
- ✅ Several directories in one mapping — `static/behaviour_test.go`
- ✅ Mock under a static prefix never reached — `static/behaviour_test.go`

## Rewrites

[Request rewriting](../../docs/Request-Rewriting.md)

- ✅ Fixed path — `rewrite/rewrite_test.go`
- ✅ `{name}` segments — `rewrite/rewrite_test.go`
- ✅ `{rest:.*}` keeps the rest of the path — `rewrite/rewrite_test.go`
- ✅ Query kept, unmatched tail dropped — `rewrite/rules_test.go`
- ✅ Method and body kept — `rewrite/rules_test.go`
- ✅ `host` with a scheme reaches only that server — `rewrite/rules_test.go`
- ✅ Rewritten path skips mocks — `rewrite/rules_test.go`
- ✅ Rewrite reported in request events — `events/events_test.go`
- ❌ `host` without a scheme uses the incoming scheme
- ❌ Cache uses the rewritten path

## Cache

[Response caching](../../docs/Response-Caching.md)

- ✅ Paths in `cache` globs served from the cache, others never — `cache/cache_test.go`
- ✅ Hit has the miss's status, headers and body — `cache/rules_test.go`
- ✅ `Age` on hits — `cache/rules_test.go`
- ✅ `expiration-time` — `cache/rules_test.go`
- ✅ CORS written per request on a hit — `cache/cache_test.go`
- ✅ Query names sorted, other queries kept apart — `cache/rules_test.go`
- ✅ `Accept-Encoding` in the key — `cache/rules_test.go`
- ✅ Not stored: non-2xx, `Set-Cookie`, `no-store`, `private`, `Vary: *` — `cache/rules_test.go`
- ✅ Method not in `cache-config.methods` not stored — `cache/rules_test.go`
- ✅ Hit reported as `HandlerCache` — `events/events_test.go`
- ✅ WebSocket never cached — `websocket/websocket_test.go`
- ❌ Headers named in upstream `Vary` in the key
- ❌ Body over 10 MiB passed through, not stored
- ❌ Upstream `Age` added to the stored age
- ❌ Cache emptied on reload

## HAR

[HAR collector](../../docs/HAR-Collector.md)

- ✅ Entry written while running — `har/har_test.go`
- ✅ Request: URL, query order, headers, JSON `postData` — `har/content_test.go`
- ✅ Response: status, headers, body — `har/content_test.go`
- ✅ Secure headers and cookies left out by default — `har/content_test.go`
- ✅ `capture-secure-headers: true` keeps them — `har/content_test.go`
- ✅ Recorded: proxied, static miss, preflight, cache hit; not: mock, script, static file — `har/content_test.go`
- ✅ Request `Content-Length` kept upstream — `har/har_test.go`
- ✅ Reload starts a new recording — `reload/reload_test.go`
- ✅ WebSocket not recorded — `websocket/websocket_test.go`
- ❌ Form, binary and over-10-MiB request bodies
- ❌ Compressed response stored decoded

## WebSocket

[Configuration: WebSocket](../../docs/Configuration.md#what-uncors-changes-in-proxied-traffic)

- ✅ Messages both ways over `ws://` and `wss://` — `websocket/websocket_test.go`
- ✅ Tunnel closed by either side — `websocket/websocket_test.go`
- ✅ Refused handshake returned as a regular response — `websocket/websocket_test.go`
- ✅ Not cached or recorded — `websocket/websocket_test.go`
- ✅ Closed on shutdown — `websocket/websocket_test.go`
- ✅ Closed on reload — `websocket/websocket_test.go`
- ✅ Outside `allowed-origins`: connects, no CORS headers — `websocket/websocket_test.go`
- ❌ Mock or script answering the handshake

## Lifecycle and reload

[Troubleshooting: configuration problems](../../docs/Troubleshooting.md#configuration-problems)

- ✅ Connections accepted during a reload — `reload/reload_test.go`
- ✅ In-flight requests finish — `reload/reload_test.go`
- ✅ Ports moved on reload — `reload/reload_test.go`
- ✅ Scheme switched on reload — `reload/reload_test.go`
- ✅ Broken file keeps the previous config — `cli/reload_test.go`
- ✅ `SIGINT` exits cleanly — `cli/reload_test.go`
- ✅ `ServerStarted` lists the mappings — `events/events_test.go`
- ✅ Every request attributed to its handler — `events/events_test.go`
- ❌ Valid file change picked up by the watcher in the binary

## Validation and errors

[Troubleshooting](../../docs/Troubleshooting.md#configuration-problems)

- ✅ YAML syntax errors with line — `cli/validation_test.go`
- ✅ Unknown keys with path and line — `cli/validation_test.go`
- ✅ Wrong value types with path and line — `cli/validation_test.go`
- ✅ Semantic checks: paths, methods, codes, files, dirs, schemes, duplicates, HAR, cache — `cli/validation_test.go`
- ✅ Bad flags — `cli/validation_test.go`
- ✅ `https` without a local CA — `cli/validation_test.go`
- ✅ Port already in use — `cli/validation_test.go`
- ✅ Error page: status, reason, CORS, no caching — `proxy/`, `script/`, `mock/`, `events/`
- ❌ `http` and `https` mappings on the same port

## CLI and tooling

[Configuration: flags](../../docs/Configuration.md#command-line-flags) ·
[Tracing](../../docs/Configuration.md#tracing-and-diagnostics)

- ✅ `--version` — `cli/cli_test.go`
- ❌ `generate-certs`
- ❌ `.uncors.yaml` / `.uncors.yml` found without `--config`
- ❌ Terminal UI (`--interactive`), checked by `testing/stories` instead
- ❌ `UNCORS_TRACE` export

## What to cover next

1. **Proxying to real upstreams:** `https://` upstream, 504 on a slow upstream,
   `to: //host`, upstream proxy settings. Every user hits these first.
2. **Config entry points:** shorthand mappings, `--from` / `--to`, `{name}` in
   `to`, default config file, a valid change picked up by the watcher.
3. **Header details:** `SameSite=None` → `Lax`, expose-headers list, upstream
   `Access-Control-Allow-Headers` removed, `304` `Content-Length`.
4. **Cache edges:** `Vary` key, 10 MiB limit, upstream `Age`, reload, rewritten
   path.
5. **The rest:** HAR form/binary/compressed bodies, rewrite `host` without a
   scheme, mock/script filters and disconnects, WebSocket handshake answered by
   a mock, `generate-certs`.

Update this file with the test that covers a feature, and add a line here when
`docs/` gains one.
