You can configure UNCORS with command-line flags, a YAML file, or both. When
you use both, flags take precedence over the file.

Everything revolves around host mappings. A mapping says "requests that arrive
at this local address go to that remote address". Global options such as the
upstream proxy apply to all mappings. Per-mapping options such as mocks or
static files apply only to the mapping they are defined in.

## Contents

- [Quick reference](#quick-reference)
- [Command-line flags](#command-line-flags)
- [Tracing and diagnostics](#tracing-and-diagnostics)
- [Configuration file](#configuration-file)
- [Global options](#global-options)
- [Mapping options](#mapping-options)
  - [Ports](#ports)
  - [Protocol scheme mapping](#protocol-scheme-mapping)
  - [Named placeholder mapping](#named-placeholder-mapping)
  - [Simplified syntax](#simplified-syntax)
  - [OPTIONS request handling](#options-request-handling)
  - [Allowed origins](#allowed-origins)
- [What UNCORS changes in proxied traffic](#what-uncors-changes-in-proxied-traffic)
- [HAR recording](#har-recording)
- [HTTPS configuration](#https-configuration)
- [Proxy configuration](#proxy-configuration)

## Quick reference

The smallest useful configuration:

```yaml
mappings:
  - from: http://api.local:3000
    to: https://api.example.com
```

A proxy with caching:

```yaml
mappings:
  - from: http://api.local:3000
    to: https://api.example.com
    cache:
      - /api/**
```

A single-page app served from a local build:

```yaml
mappings:
  - from: http://app.local:3000
    to: https://app.example.com
    statics:
      - path: /
        dir: ./dist
        index: index.html
```

A proxy with one mocked endpoint:

```yaml
mappings:
  - from: http://api.local:3000
    to: https://api.example.com
    mocks:
      - path: /api/test
        response:
          code: 200
          headers:
            Content-Type: application/json
          raw: '{"status": "ok"}'
```

> [!TIP]
> [Real-World Examples](Real-World-Examples) has complete configurations for
> common setups.

## Command-line flags

| Flag            | Short | Description                                                                                       |
| --------------- | ----- | ------------------------------------------------------------------------------------------------- |
| `--from`        | `-f`  | Source URL with scheme and port, for example `http://localhost:8080`. Can be repeated.            |
| `--to`          | `-t`  | Target URL with scheme, for example `https://api.example.com`. Can be repeated.                   |
| `--config`      | `-c`  | Path to a YAML configuration file.                                                                |
| `--proxy`       |       | HTTP/HTTPS proxy for upstream requests. Overrides `proxy` from the file.                          |
| `--interactive` |       | Run the interactive terminal UI. Defaults to `true`; pass `--interactive=false` for plain output. |
| `--version`     | `-v`  | Print the version and exit.                                                                       |
| `--help`        | `-h`  | Print usage and exit.                                                                             |

`--from` and `--to` work in pairs: the first `--from` goes with the first
`--to`, and so on. UNCORS refuses to start if the counts differ. Each pair
becomes a mapping. If a `--from` value is the same as a `from` already in the
configuration file, the flag replaces that mapping's `to` instead of adding a
new mapping. The comparison ignores case, a missing `http://`, and the default
port, so `--from localhost:8080` replaces `from: http://localhost:8080`.

```bash
uncors --from http://localhost:8080 --to https://api.example.com \
       --from http://localhost:8081 --to https://auth.example.com
```

To generate the local CA used for HTTPS mappings, run
`uncors generate-certs`. See [HTTPS configuration](#https-configuration).

## Tracing and diagnostics

UNCORS always prints handled requests to the console. Internal diagnostics are
off by default. To see them, set `UNCORS_TRACE=true`: UNCORS then exports
[OpenTelemetry](https://opentelemetry.io/) traces, logs, and metrics over
OTLP/HTTP to a collector. Without the variable nothing is collected, and UNCORS
makes no connection to a collector.

The exporters read the standard `OTEL_EXPORTER_OTLP_*` environment variables.
The endpoint defaults to `http://localhost:4318`; set
`OTEL_EXPORTER_OTLP_ENDPOINT` to send elsewhere, or the `_TRACES_`, `_LOGS_`,
and `_METRICS_` variants to split the signals. Headers, timeouts, and TLS
settings use the other `OTEL_EXPORTER_OTLP_*` variables. Metrics are exported
every 10 seconds unless `OTEL_METRIC_EXPORT_INTERVAL` (in milliseconds) says
otherwise. `OTEL_LOGS_EXPORTER=none` and `OTEL_METRICS_EXPORTER=none` turn a
signal off.

`UNCORS_TRACE_CONNECTIONS=true` also records the DNS, connection, TLS, and
header phases of each upstream request as events on its client span. It is off
by default because it nearly doubles the cost of tracing an upstream request;
turn it on to find out why an upstream is slow.

To see everything locally, run [otelop](https://github.com/mashiro/otelop) and
open <http://localhost:4319>:

```bash
docker run --rm -p 4318:4318 -p 4319:4319 ghcr.io/mashiro/otelop
UNCORS_TRACE=true uncors --config .uncors.yaml
```

Jaeger works too, for traces only:

```bash
docker run --rm -p 16686:16686 -p 4318:4318 jaegertracing/jaeger
UNCORS_TRACE=true OTEL_LOGS_EXPORTER=none OTEL_METRICS_EXPORTER=none \
  uncors --config .uncors.yaml
```

Without the two `none` settings UNCORS warns that Jaeger rejects logs and
metrics.

### Traces

Each request is one trace. Its root span, `HTTP <method>`, records:

- the request: `uncors.request.id` (the number shown in the request log), URL,
  scheme, path, host and port, client address, protocol version, user agent,
  body size, and the `Origin`, `Referer`, `Content-Type`,
  `Access-Control-Request-*`, `X-Request-Id`, and `X-Correlation-Id` headers;
- the route that matched: the mapping (`uncors.mapping.from` and `.to`), the
  route kind (`static`, `mock`, `script`, `rewrite`, or `default`), the route
  pattern (`http.route`) and its position in the configuration, and the method,
  query, and header filters;
- the result: the handler that answered (`uncors.handler`), the rewrite, the
  status, the response body size, the `Content-Type`, `Location`,
  `Cache-Control`, and `Access-Control-Allow-*` response headers, and the
  error, if any.

Credentials and cookies are never recorded. While tracing is on, every
response carries an `X-Uncors-Trace-Id` header with the trace ID, so a request
seen in the browser's developer tools can be looked up in the trace viewer.

Child spans show the work done for the request:

| Span                                                       | What it records                                                                          |
| ---------------------------------------------------------- | ---------------------------------------------------------------------------------------- |
| `static.serve`                                             | The file asked for and the file served, its size, the index fallback, or a fall-through. |
| `mock.serve`                                               | The configured status, delay, and source; events when the delay starts and ends.         |
| `script.serve`, `script.lua.compile`, `script.lua.execute` | The script source and size, the status it wrote, and the Lua stack trace of an error.    |
| `rewrite.apply`                                            | The pattern, template, path variables, original and new path, and target host.           |
| `har.record`                                               | The HAR file, body sizes, and whether the entry was dropped because the queue was full.  |
| `cache.lookup`, `cache.store`                              | The cache key, hit or miss, the cached status and size, and whether the store declined.  |
| `options.preflight`                                        | An answered `OPTIONS` request and its configured status.                                 |
| `proxy.forward`, `proxy.copy_body`                         | The upstream URL, its content type, length, and protocol, and the bytes copied.          |
| `HTTP <method>` (client)                                   | The real upstream request; its connection phases with `UNCORS_TRACE_CONNECTIONS`.        |

Requests that skip the cache add a `uncors.cache.not_cacheable` event, and
responses that are not stored add `uncors.cache.not_stored`, or
`uncors.cache.rejected` when the cache storage declined them. Lifecycle work has
its own traces: `engine.start`, `engine.restart`, `engine.shutdown`,
`supervisor.reload`, `config.load`, `server.listen`, `tls.get_certificate`
(with the SNI name and ALPN protocols), `tls.ensure_ca` (with the CA expiry),
`tls.generate_certificate`, `har.flush`, and `version.check`.

Trace context is never added to proxied requests: the upstream server receives
the request as the client sent it.

### Logs

Logs are exported with the trace and span they belong to, so a trace viewer
shows them next to the request that produced them.

| Level   | Logged                                                                                                                                                                                            |
| ------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ERROR` | A request a handler failed (method, URL, error); a HAR file that cannot be written.                                                                                                               |
| `WARN`  | An upstream `5xx` response; a Lua script error, with its stack trace; a HAR entry dropped because the write queue is full.                                                                        |
| `DEBUG` | Each routing decision: the upstream URL and status, cache hits, misses, and stores, answered preflights, rewrites, static files served or passed on, mock responses, script runs, and HAR writes. |

A request the client cancels is logged at `DEBUG`, not as a failure.

### Metrics

| Metric                                                                     | What it counts                                                       |
| -------------------------------------------------------------------------- | -------------------------------------------------------------------- |
| `uncors.requests`                                                          | Requests received, by method, status, and the handler that answered. |
| `http.server.request.duration`, `http.server.response.body.size`           | Time to serve a request and the size of the response.                |
| `http.server.active_requests`                                              | Requests being served.                                               |
| `http.client.request.duration` and the other `http.client.*` metrics       | Upstream requests by host and status or error type, timed and sized. |
| `uncors.cache.lookups`, `uncors.cache.stores`, `uncors.cache.stored_bytes` | Cache hits and misses, and responses stored.                         |
| `uncors.cache.rejections`                                                  | Responses the cache storage declined to store.                       |
| `uncors.mock.responses`, `uncors.mock.delay`                               | Mock responses, served or cancelled during their delay.              |
| `uncors.script.executions`, `uncors.script.duration`                       | Lua script runs, by success or error, and their duration.            |
| `uncors.static.requests`                                                   | Static requests served, passed on to the proxy, or failed.           |
| `uncors.rewrites`                                                          | Requests rewritten.                                                  |
| `uncors.har.entries`, `uncors.har.flushes`                                 | HAR entries recorded or dropped, and HAR file writes.                |
| `uncors.config.reloads`, `uncors.tls.certificates`                         | Configuration reloads and host certificates generated.               |
| `go.memory.*`, `go.goroutine.count`, and the other Go runtime metrics      | The UNCORS process itself.                                           |

### Console output

Diagnostics never appear in the console. When tracing is on, UNCORS shows one
notice with the endpoint at startup, and one warning for each signal (spans,
logs, or metrics) that cannot be exported, for example because no collector is
listening. When it exits, UNCORS waits up to 5 seconds for the last data to be
sent.

Failures that need your attention are shown in the console whether tracing is
on or not:

- the HAR file cannot be written (shown once until a write succeeds again);
- the config file watcher reports an error;
- a TLS certificate cannot be loaded or generated (once for the CA, once per
  host);
- a client asks for a TLS certificate for a host that no `https://` mapping
  covers (once per host);
- in-flight requests are cut off because they did not finish before a restart.

## Configuration file

The configuration file is YAML. A JSON Schema is available at
[`schema.json`](https://raw.githubusercontent.com/evg4b/uncors/main/schema.json);
editors with a YAML language server can use it for completion and validation
if you add this line at the top of the file:

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/evg4b/uncors/main/schema.json
```

An example that uses most options:

```yaml
proxy: http://localhost:8888

cache-config:
  expiration-time: 10m
  max-size: 52428800
  methods: [GET]

mappings:
  - http://localhost:8080: https://github.com
  - from: http://other.local:3000
    to: https://example.com
    statics:
      /static: ./public
      /docs: ./build/docs
    mocks:
      - path: /hello
        response:
          code: 200
          delay: 1m30s
          raw: Hello world
      - path: /world
        method: POST
        response:
          code: 203
          delay: 5s
          file: ./mocks/world.json
    scripts:
      - path: /api/custom
        method: POST
        script: |
          local json = require("json")
          local data = json.decode(request.body)
          response.headers["Content-Type"] = "application/json"
          response:WriteHeader(200)
          response:WriteString(json.encode({status = "ok", received = data}))
    cache:
      - /api/**
    har: ./recordings/other.har
```

Relative paths (`dir`, `file`, `har`) are resolved against the directory you
start UNCORS from, not the directory of the configuration file. `~` is not
expanded, so use a relative or absolute path instead.

UNCORS validates the whole file at startup and reports every problem it finds,
with the path to the bad value (for example
`mappings[0].mocks[0].response.code code must be in range 100-599`). It also
checks that the files and directories referenced by mocks, scripts, and statics
exist.

A key that UNCORS does not know is an error at any level of the file, so a typo
does not silently turn an option off:

```text
mappings[0].mocks[0].response.cod: unknown field (line 8)
```

While UNCORS runs, it watches the configuration file. When the file changes, it
loads and validates the new version and restarts the server with it. If the new
version is invalid, UNCORS prints the error and keeps running with the previous
configuration. A restart clears the response cache and starts new HAR
recordings.

## Global options

| Option         | Type   | Default | Description                                                                                 |
| -------------- | ------ | ------- | ------------------------------------------------------------------------------------------- |
| `mappings`     | array  | -       | Host mappings (see below). At least one mapping is required, from the file or from flags.   |
| `proxy`        | string | -       | HTTP/HTTPS proxy URL for upstream requests, including the scheme. See [Proxy configuration](#proxy-configuration). |
| `cache-config` | object | -       | Cache settings shared by all mappings. See [Response Caching](Response-Caching).            |

## Mapping options

Each entry under `mappings` has a source and a target, plus optional features:

| Option             | Type             | Description                                                                         |
| ------------------ | ---------------- | ----------------------------------------------------------------------------------- |
| `from`             | string           | Required. Local URL that UNCORS listens on, for example `http://localhost:8080`.    |
| `to`               | string           | Required. Upstream URL that requests are forwarded to.                              |
| `mocks`            | array            | Predefined responses. See [Response Mocking](Response-Mocking).                     |
| `scripts`          | array            | Lua handlers. See [Script Handler](Script-Handler).                                 |
| `statics`          | array or map     | Local directories to serve. See [Static File Serving](Static-File-Serving).         |
| `cache`            | array of strings | Path globs whose upstream responses are cached. See [Response Caching](Response-Caching). |
| `rewrites`         | array            | Path and host rewrites. See [Request Rewriting](Request-Rewriting).                 |
| `options-handling` | object           | How `OPTIONS` requests are answered. See [below](#options-request-handling).        |
| `allowed-origins`  | array of strings | Origins that get CORS headers. See [below](#allowed-origins).                       |
| `har`              | string or object | Record traffic to a HAR file. See [HAR Collector](HAR-Collector).                   |

`from` and `to` contain only a scheme, a host, and an optional port. A path or
query string in either is a configuration error.

### Ports

The port in the `from` URL is the port UNCORS listens on. Without a port,
UNCORS uses 80 for `http` and 443 for `https`.

Several mappings can share a port; UNCORS picks the mapping by the request's
host name. Mappings on different ports each get their own listener:

```yaml
mappings:
  - from: http://api.local:3000
    to: https://api.example.com
  - from: http://auth.local:3000
    to: https://auth.example.com
  - from: http://admin.local:4000
    to: https://admin.example.com
```

Mappings that share a port must use the same scheme: `http://localhost:3000`
and `https://localhost:3000` in one file is a configuration error. Two mappings
with the same `from` are an error too. UNCORS compares them after adding the
default scheme and port and ignores the case of the host name, so
`localhost:80` and `http://LOCALHOST` are the same `from`.

### Protocol scheme mapping

The `from` and `to` schemes are independent, so UNCORS can serve HTTP locally
for an HTTPS upstream, or the other way round:

```yaml
mappings:
  # Local HTTP, upstream HTTPS
  - from: http://localhost:8080
    to: https://site.com

  # Local HTTPS, upstream HTTP
  - from: https://localhost:8443
    to: http://site.com
```

A `to` URL can leave the scheme out by starting with `//`. UNCORS then uses the
scheme of the incoming request:

```yaml
mappings:
  - from: http://localhost:8080
    to: //site.com # requests go to http://site.com
  - from: https://localhost:8443
    to: //site.com # requests go to https://site.com
```

A `from` URL without a scheme (for example `//localhost:8080`) is treated as
`http://`.

> [!NOTE]
> HTTPS sources need a local CA. See [HTTPS configuration](#https-configuration).

### Named placeholder mapping

A host name in `from` can contain placeholders written as `{name}`. A
placeholder matches one label of the host name, that is, any text without a
dot. The same name in `to` is replaced with the matched value.

A placeholder source with a fixed target sends every matching host to the same
place:

```yaml
mappings:
  - from: http://{repo}.local.com:8080
    to: https://github.com
```

| Local request                           | Upstream request                |
| --------------------------------------- | ------------------------------- |
| `http://raw.local.com:8080`             | `https://github.com`            |
| `http://raw.local.com:8080/api/info`    | `https://github.com/api/info`   |
| `http://docs.local.com:8080/index.html` | `https://github.com/index.html` |

Using the placeholder in the target carries the subdomain over:

```yaml
mappings:
  - from: http://{repo}.local.com:8080
    to: https://{repo}.github.com
```

| Local request                           | Upstream request                     |
| --------------------------------------- | ------------------------------------ |
| `http://raw.local.com:8080`             | `https://raw.github.com`             |
| `http://raw.local.com:8080/api/info`    | `https://raw.github.com/api/info`    |
| `http://docs.local.com:8080/index.html` | `https://docs.github.com/index.html` |

With several placeholders, values are matched by name, so their order can
differ between source and target:

```yaml
mappings:
  - from: http://{env}.{service}.local.com
    to: https://{service}.{env}.api.com
```

| Local request                      | Upstream request                  |
| ---------------------------------- | --------------------------------- |
| `http://prod.auth.local.com`       | `https://auth.prod.api.com`       |
| `http://prod.auth.local.com/login` | `https://auth.prod.api.com/login` |
| `http://staging.users.local.com`   | `https://users.staging.api.com`   |

Placeholder names start with a letter and contain only letters, digits, and
underscores. Each name can appear only once in a `from` URL; `{client}.{client}.com`
is a configuration error. The `*` wildcard from older versions is no longer
accepted; see the [Migration Guide](Migration-Guide).

> [!WARNING]
> UNCORS only receives requests for host names that resolve to your machine.
> The hosts file has no wildcards, so `http://{name}.local.com` only works for
> the subdomains you list there one by one.

### Simplified syntax

A mapping that needs nothing but `from` and `to` can be written as a single
`from: to` pair. The key must look like a host: it contains a scheme, a port,
or a dot, or is `localhost`. Any other single key is read as an option of the
full form, so a typo such as `- frm: http://localhost:3000` is reported as an
unknown field.

```yaml
mappings:
  - http://localhost:8080: https://github.com
```

Both forms can be mixed in one file:

```yaml
mappings:
  - http://localhost:8080: https://github.com
  - http://host1:3000: https://gitlab.com
  - http://{repo}.local:8080: https://{repo}.example.io
  - from: http://host2:9090
    to: https://gitea.com
    mocks:
      - path: /api/ping
        response:
          code: 200
          raw: pong
```

### OPTIONS request handling

UNCORS answers `OPTIONS` requests itself by default, so CORS preflight checks
pass without reaching the upstream server. The default response has status 200
and these headers:

| Header                             | Value                                                                                                         |
| ---------------------------------- | ------------------------------------------------------------------------------------------------------------- |
| `Access-Control-Allow-Origin`      | The request's `Origin`, or `*` if there is none                                                               |
| `Access-Control-Allow-Methods`     | The request's `Access-Control-Request-Method`, or `GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS`              |
| `Access-Control-Allow-Headers`     | The request's `Access-Control-Request-Headers`, or `*`                                                        |
| `Access-Control-Allow-Credentials` | `true`                                                                                                        |
| `Access-Control-Expose-Headers`    | `*`                                                                                                           |
| `Access-Control-Max-Age`           | `86400`                                                                                                       |
| `Vary`                             | The request headers copied above: `Origin`, `Access-Control-Request-Method`, `Access-Control-Request-Headers` |

The same headers are added to `OPTIONS` requests forwarded to the upstream
server when `disabled` is `true`.

| Option     | Type    | Default | Description                                                  |
| ---------- | ------- | ------- | ------------------------------------------------------------ |
| `disabled` | boolean | `false` | Forward `OPTIONS` requests to the upstream server instead.   |
| `code`     | integer | `200`   | Status code of the `OPTIONS` response.                       |
| `headers`  | object  | -       | Extra headers. They replace the defaults when names collide. |

```yaml
mappings:
  - from: http://localhost:8080
    to: https://github.com
    options-handling:
      code: 204
      headers:
        Access-Control-Allow-Origin: http://localhost
        Access-Control-Allow-Methods: GET, POST
```

To let the upstream server answer preflight requests:

```yaml
mappings:
  - from: http://localhost:8080
    to: https://github.com
    options-handling:
      disabled: true
```

### Allowed origins

By default UNCORS copies any `Origin` into `Access-Control-Allow-Origin` and
sends `Access-Control-Allow-Credentials: true`. Any page open in your browser
can then call the mapping with your cookies and read the response. See
[Any website can read responses through UNCORS](Troubleshooting#any-website-can-read-responses-through-uncors).

`allowed-origins` limits the origins that get CORS headers. Each entry is a
glob pattern matched against the whole `Origin` header:

| Pattern                        | Matches                                                 |
| ------------------------------ | ------------------------------------------------------- |
| `http://localhost:*`           | `http://localhost` on any port                          |
| `https://*.example.test`       | Any subdomain of `example.test` over HTTPS              |
| `http://127.0.0.1:{3000,5173}` | `http://127.0.0.1:3000` and `http://127.0.0.1:5173`     |
| `null`                         | The `null` origin of sandboxed frames and `file:` pages |

`*` matches any characters except `/`, `?` matches one character, `[abc]` one
of the listed characters, and `{a,b}` one of the alternatives. Browsers send
the origin as scheme, host, and port only, in lowercase, so a pattern with a
path or a trailing slash never matches; UNCORS rejects a pattern that ends with
`/`.

When `allowed-origins` is set and a request has an `Origin` that matches none
of the patterns:

- the request is still handled: it is proxied, mocked, scripted, or served
  from the cache as usual;
- the response has no `Access-Control-*` headers, including those sent by the
  upstream server, a mock, or a script, so the browser does not let the page
  read it. `Vary: Origin` is added;
- a preflight request answered by UNCORS gets `403 Forbidden` without CORS
  headers or the `options-handling.headers`. With `options-handling.disabled`,
  the preflight is forwarded and the upstream response loses its CORS headers.

The rule covers proxied, mock, script, and cached responses, preflight
responses, and error pages. Requests without an `Origin` header, such as
navigations and same-origin `GET` requests, get CORS headers as before. Without
`allowed-origins` any origin gets CORS headers.

```yaml
mappings:
  - from: http://api.local:3000
    to: https://api.example.com
    allowed-origins:
      - http://localhost:*
      - http://app.local:3000
```

## What UNCORS changes in proxied traffic

For requests forwarded to the upstream server, UNCORS:

- passes the path and query on as they came: repeated slashes, `.` and `..`
  segments, and escaped characters such as `%2F` are not cleaned or decoded;
- rewrites the `Origin` and `Referer` request headers from the local host to the
  upstream host. An `Origin` or `Referer` of any other host (for example a
  frontend served from `127.0.0.1` or another local domain), and
  `Origin: null`, is forwarded unchanged;
- forwards the request body with its `Content-Length`;
- waits up to 5 minutes for the upstream server to start its response (30
  seconds to connect, 10 seconds for the TLS handshake). The response body has
  no time limit: it is forwarded until the upstream server finishes it or the
  client disconnects;
- passes event streams (`text/event-stream`) and responses without a
  `Content-Length` on to the client as each chunk arrives, so server-sent events
  and long downloads are not held back;
- forwards the `Cookie` request header unchanged, and rewrites the `Domain` of
  cookies set by the upstream server to the local host. The `Secure` flag of
  those cookies follows the scheme of the local host;
- rewrites `Location` headers in redirects back to the local host;
- adds the CORS headers listed below to every response, replacing any the
  upstream server sent, unless the origin is not in
  [`allowed-origins`](#allowed-origins). `Access-Control-Allow-Headers` from
  the upstream server is removed: it only matters in preflight responses.

| Header                             | Value                                                                                            |
| ---------------------------------- | ------------------------------------------------------------------------------------------------ |
| `Access-Control-Allow-Origin`      | The request's `Origin`, or `*` if there is none                                                  |
| `Access-Control-Allow-Credentials` | `true`                                                                                           |
| `Access-Control-Allow-Methods`     | `GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS`                                                   |
| `Access-Control-Expose-Headers`    | `Date` and the names of the other response headers if the request has an `Origin`, or `*`        |
| `Access-Control-Max-Age`           | `86400`                                                                                          |
| `Vary`                             | `Origin` is added when the request has an `Origin`; the upstream server's `Vary` values are kept |

Requests sent with credentials (cookies or `Authorization`) need the exact
origin and an explicit list of exposed headers: browsers read `*` in
`Access-Control-Expose-Headers` literally for them. The list leaves out the
headers JavaScript can always read (`Cache-Control`, `Content-Language`,
`Content-Length`, `Content-Type`, `Expires`, `Last-Modified`, `Pragma`), the
`Access-Control-*` headers, and `Set-Cookie`.

Mock and script responses get the same CORS headers. For them, the list of
exposed headers is built when the status is written, so it includes the
headers set by the mock or the script. An `Access-Control-Expose-Headers` set
by the mock or the script is kept. Files served by
[statics](Static-File-Serving) get no CORS headers.

## HAR recording

UNCORS can record traffic for a mapping to an
[HTTP Archive (HAR 1.2)](https://w3c.github.io/web-performance/specs/HAR/Overview.html)
file that you can open in browser DevTools or other HAR viewers.

Give the output path as a string:

```yaml
mappings:
  - from: http://api.local:3000
    to: https://api.example.com
    har: ./recordings/api.har
```

Or use an object for more control:

```yaml
mappings:
  - from: http://api.local:3000
    to: https://api.example.com
    har:
      file: ./recordings/api.har
      capture-secure-headers: false # default
```

| Option                   | Type    | Default | Description                                                                                         |
| ------------------------ | ------- | ------- | --------------------------------------------------------------------------------------------------- |
| `file`                   | string  | -       | Output file. It must have an extension, such as `.har`.                                             |
| `capture-secure-headers` | boolean | `false` | Keep cookies and auth headers in the recording (see [HAR Collector](HAR-Collector#secure-headers)). |

> [!WARNING]
> With `capture-secure-headers: true`, tokens and cookies are written to disk in
> plain text. Don't commit those files.

[HAR Collector](HAR-Collector) has the details, including which requests are
recorded.

## HTTPS configuration

UNCORS can serve HTTPS on the local side. It signs a certificate for each host
on the fly with a local certificate authority (CA) that you create once.

### Create and trust the local CA

```bash
# Create the CA (valid for 365 days by default)
uncors generate-certs

# Choose a different validity period
uncors generate-certs --validity-days 730

# Replace an existing CA
uncors generate-certs --force
```

This writes two files to `~/.config/uncors/`:

- `ca.crt`, the CA certificate, which you add to your system's trust store;
- `ca.key`, its private key, readable only by you (`0600`), also when
  `--force` replaces an existing file.

Without `--force`, the command stops if either file already exists.
`--validity-days` must be a positive number.

Trust `ca.crt` so browsers accept the certificates UNCORS creates:

- macOS: open `ca.crt`, add it to Keychain Access, and set it to Always Trust.
- Linux: copy it to `/usr/local/share/ca-certificates/` and run
  `sudo update-ca-certificates`.
- Windows: import it into Trusted Root Certification Authorities with the
  Certificate Manager (`certmgr.msc`).

Firefox uses its own certificate store; see
[Troubleshooting](Troubleshooting#https-certificate-errors).

### Use HTTPS mappings

```yaml
mappings:
  - from: https://localhost:8443
    to: https://github.com
```

UNCORS refuses to start if a mapping uses `https://` in `from` and the CA files
are missing. The HTTPS listener only starts when at least one mapping uses an
`https://` source.

UNCORS signs a certificate only for the host names of the `https://` mappings
in the current configuration, and keeps it in memory. It reads the host name
the client asks for (through SNI). A `{name}` placeholder matches one label, as
in routing: `https://{tenant}.local.dev` covers `acme.local.dev` but not
`a.b.local.dev`. For any other host name the TLS handshake fails, and UNCORS
shows a notice once per host. When the configuration is reloaded, the new host
names take effect.

A client that connects to an IP address sends no host name. UNCORS then uses
the address the client reached (`127.0.0.1`), so this works only with a
mapping such as `https://127.0.0.1:8443`. Otherwise, connect by host name.

Host certificates use ECDSA P-256 keys and are valid for a year, but never
past the expiry date of the CA.

## Proxy configuration

UNCORS can send upstream requests through an HTTP or HTTPS proxy.

By default it uses the standard proxy environment variables:

- `HTTP_PROXY` / `http_proxy`
- `HTTPS_PROXY` / `https_proxy`
- `NO_PROXY` / `no_proxy`

Values can be a full URL (`http://proxy.example.com:8080`) or just a host and
port (`proxy.example.com:8080`), which is treated as HTTP.

Setting a proxy explicitly overrides the environment variables. In the
configuration file or on the command line, the value must be a full URL with a
scheme:

```yaml
proxy: http://proxy.example.com:8080
```

```bash
uncors --proxy http://proxy.example.com:8080 --from http://localhost --to https://api.example.com
```

For a proxy that needs authentication, put the credentials in the URL:
`http://username:password@proxy.example.com:8080`.
