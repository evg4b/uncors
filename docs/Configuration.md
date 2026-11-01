You can configure UNCORS with command-line flags, a YAML file, or both. Flags
take precedence over the file.

Everything is built around host mappings. A mapping sends requests that arrive
at a local address to a remote one. Global options, such as the upstream proxy,
apply to all mappings; per-mapping options, such as mocks or static files,
apply only to the mapping they are defined in.

## Contents

- [Quick reference](#quick-reference)
- [Command-line flags](#command-line-flags)
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
- [HTTPS configuration](#https-configuration)
- [Proxy configuration](#proxy-configuration)
- [Tracing and diagnostics](#tracing-and-diagnostics)

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
| `--config`      | `-c`  | Path to a YAML configuration file. Defaults to `.uncors.yaml` or `.uncors.yml` if one exists.     |
| `--proxy`       |       | HTTP/HTTPS proxy for upstream requests. Overrides `proxy` from the file.                          |
| `--interactive` |       | Run the interactive terminal UI. Defaults to `true`; pass `--interactive=false` for plain output. |
| `--version`     | `-v`  | Print the version, Go version, and commit, then exit.                                             |
| `--help`        | `-h`  | Print usage and exit.                                                                             |

`--from` and `--to` work in pairs: the first `--from` goes with the first
`--to`, and so on. UNCORS refuses to start if the counts differ.

```bash
uncors --from http://localhost:8080 --to https://api.example.com \
       --from http://localhost:8081 --to https://auth.example.com
```

> [!NOTE]
> If a `--from` matches a `from` already in the configuration file, the flag
> replaces that mapping's `to` instead of adding a mapping. Case, a missing
> `http://`, and the default port are ignored, so `--from localhost:8080`
> matches `from: http://localhost:8080`. This also applies to a file found
> without `--config`.

`uncors generate-certs` creates the local CA for HTTPS mappings. See
[HTTPS configuration](#https-configuration).

## Configuration file

Without `--config`, UNCORS loads `.uncors.yaml` or, failing that,
`.uncors.yml` from the current directory. If neither exists, it runs with the
flags alone.

A JSON Schema is available for editors with a YAML language server. Add this
line at the top of the file to get completion and validation:

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
expanded.

UNCORS validates the whole file at startup and reports every problem with the
path to the bad value, for example
`mappings[0].mocks[0].response.code must be in range 200-599`. Files and
directories used by mocks, scripts, and statics must exist. An unknown key is
an error at any level, so a typo cannot silently turn an option off:

```text
mappings[0].mocks[0].response.cod: unknown field (line 8)
```

### Reloading

UNCORS watches the configuration file and restarts with the new version when
it changes. If the new version is invalid, UNCORS prints the error and keeps
the previous configuration. A reload clears the response cache and starts new
HAR recordings.

> [!NOTE]
> A reload does not drop connections on ports that stay in the file with the
> same scheme. Requests already in progress finish with the old configuration;
> any still running after 15 seconds are cut off with a warning. Ports removed
> from the file stop accepting connections, and a port that switches between
> `http://` and `https://` is opened again with the new scheme.

## Global options

| Option         | Type   | Description                                                                                            |
| -------------- | ------ | ------------------------------------------------------------------------------------------------------ |
| `mappings`     | array  | Host mappings. At least one is required, from the file or from flags.                                  |
| `proxy`        | string | HTTP/HTTPS proxy URL for upstream requests. See [Proxy configuration](#proxy-configuration).           |
| `cache-config` | object | Cache settings shared by all mappings. See [Response Caching](Response-Caching#global-cache-settings). |

## Mapping options

| Option             | Type             | Description                                                                               |
| ------------------ | ---------------- | ----------------------------------------------------------------------------------------- |
| `from`             | string           | Required. Local URL that UNCORS listens on, for example `http://localhost:8080`.          |
| `to`               | string           | Required. Upstream URL that requests are forwarded to.                                    |
| `mocks`            | array            | Predefined responses. See [Response Mocking](Response-Mocking).                           |
| `scripts`          | array            | Lua handlers. See [Script Handler](Script-Handler).                                       |
| `statics`          | array or map     | Local directories to serve. See [Static File Serving](Static-File-Serving).               |
| `cache`            | array of strings | Path globs whose upstream responses are cached. See [Response Caching](Response-Caching). |
| `rewrites`         | array            | Path and host rewrites. See [Request Rewriting](Request-Rewriting).                       |
| `options-handling` | object           | How `OPTIONS` requests are answered. See [below](#options-request-handling).              |
| `allowed-origins`  | array of strings | Origins that get CORS headers. See [below](#allowed-origins).                             |
| `har`              | string or object | Record traffic to a HAR file. See [HAR Collector](HAR-Collector).                         |

`from` and `to` contain only a scheme, a host, and an optional port. A path or
query string in either is an error.

### Ports

The port in `from` is the port UNCORS listens on: 80 for `http` and 443 for
`https` when none is given. Several mappings can share a port, and UNCORS picks
one by the request's host name:

```yaml
mappings:
  - from: http://api.local:3000
    to: https://api.example.com
  - from: http://auth.local:3000
    to: https://auth.example.com
  - from: http://admin.local:4000
    to: https://admin.example.com
```

Mappings on one port must use the same scheme, and two mappings cannot have the
same `from`.

> [!NOTE]
> UNCORS listens on `127.0.0.1` only, so a `from` with another IP address, such
> as `[::1]`, is an error. Node.js 17 and later may resolve `localhost` to
> `::1`; use `127.0.0.1` in the client or map the host name to `127.0.0.1` in
> the hosts file.

### Protocol scheme mapping

The `from` and `to` schemes are independent:

```yaml
mappings:
  # Local HTTP, upstream HTTPS
  - from: http://localhost:8080
    to: https://site.com

  # Local HTTPS, upstream HTTP
  - from: https://localhost:8443
    to: http://site.com
```

A `to` that starts with `//` uses the scheme of the incoming request. A `from`
without a scheme is treated as `http://`.

```yaml
mappings:
  - from: http://localhost:8080
    to: //site.com # requests go to http://site.com
  - from: https://localhost:8443
    to: //site.com # requests go to https://site.com
```

HTTPS sources need a local CA. See [HTTPS configuration](#https-configuration).

### Named placeholder mapping

A host name in `from` can contain placeholders written as `{name}`. A
placeholder matches one label of the host name (text without a dot). The same
name in `to` is replaced with the matched value. Values are matched by name, so
their order can differ:

```yaml
mappings:
  - from: http://{repo}.local.com:8080
    to: https://{repo}.github.com
  - from: http://{env}.{service}.local.com
    to: https://{service}.{env}.api.com
```

| Local request                        | Upstream request                  |
| ------------------------------------ | --------------------------------- |
| `http://raw.local.com:8080/api/info` | `https://raw.github.com/api/info` |
| `http://prod.auth.local.com/login`   | `https://auth.prod.api.com/login` |
| `http://staging.users.local.com`     | `https://users.staging.api.com`   |

A placeholder that is not used in `to` just matches; every matching host goes
to the same target.

Names start with a letter and contain letters, digits, and underscores. Each
name can appear only once in a `from`, so `{client}.{client}.com` is an error.
The `*` wildcard from older versions is no longer accepted; see the
[Migration Guide](Migration-Guide).

> [!WARNING]
> The hosts file has no wildcards, so `http://{name}.local.com` only receives
> requests for the subdomains you list there one by one.

### Simplified syntax

A mapping with only `from` and `to` can be written as one `from: to` pair. The
key must look like a host: it has a scheme, a port, or a dot, or is
`localhost`. Any other single key is read as an option of the full form, so a
typo such as `- frm: http://localhost:3000` is reported as an unknown field.
Both forms can be mixed:

```yaml
mappings:
  - http://localhost:8080: https://github.com
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

UNCORS answers `OPTIONS` requests itself, so CORS preflight checks pass without
reaching the upstream server. The default response has status 200 and these
headers:

| Header                             | Value                                                                                            |
| ---------------------------------- | ------------------------------------------------------------------------------------------------ |
| `Access-Control-Allow-Origin`      | The request's `Origin`, or `*`                                                                   |
| `Access-Control-Allow-Methods`     | The request's `Access-Control-Request-Method`, or `GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS` |
| `Access-Control-Allow-Headers`     | The request's `Access-Control-Request-Headers`, or `*`                                           |
| `Access-Control-Allow-Credentials` | `true`                                                                                           |
| `Access-Control-Expose-Headers`    | `*`                                                                                              |
| `Access-Control-Max-Age`           | `86400`                                                                                          |

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

UNCORS then adds the same CORS headers to the forwarded responses.

> [!NOTE]
> A mock or script answers `OPTIONS` only when it has `method: OPTIONS`, or
> when `disabled` is `true`. Otherwise UNCORS answers the preflight for its
> path as above.

### Allowed origins

By default UNCORS copies any `Origin` into `Access-Control-Allow-Origin` and
allows credentials, so any page open in your browser can call the mapping with
your cookies and read the response. `allowed-origins` limits CORS headers to
the origins you list:

```yaml
mappings:
  - from: http://api.local:3000
    to: https://api.example.com
    allowed-origins:
      - http://localhost:*
      - http://app.local:3000
```

Each entry is a glob matched against the whole `Origin` header:

| Pattern                        | Matches                                                 |
| ------------------------------ | ------------------------------------------------------- |
| `http://localhost:*`           | `http://localhost` on any port                          |
| `https://*.example.test`       | Any subdomain of `example.test` over HTTPS              |
| `http://127.0.0.1:{3000,5173}` | `http://127.0.0.1:3000` and `http://127.0.0.1:5173`     |
| `null`                         | The `null` origin of sandboxed frames and `file:` pages |

`*` matches any characters except `/`, `?` one character, `[abc]` one of the
listed characters, and `{a,b}` one of the alternatives. Browsers send the
origin as scheme, host, and port only, in lowercase, so a pattern ending in
`/` is rejected.

For an origin that matches none of the patterns, the request is still handled
as usual, but the response has no `Access-Control-*` headers, so the browser
does not let the page read it. A preflight answered by UNCORS gets
`403 Forbidden`. Requests without an `Origin`, such as navigations, are not
affected. See also
[Any website can read responses through UNCORS](Troubleshooting#any-website-can-read-responses-through-uncors).

<details>
<summary>Details for origins outside the list</summary>

- The rule covers proxied, mock, script, and cached responses, preflight
  responses, and error pages.
- `Access-Control-*` headers sent by the upstream server, a mock, or a script
  are removed too, and `Vary: Origin` is added.
- The `403` preflight carries neither CORS headers nor
  `options-handling.headers`. With `options-handling.disabled`, the preflight
  is forwarded and the upstream response loses its CORS headers.

</details>

> [!NOTE]
> `allowed-origins` does not block WebSocket connections, because browsers do
> not apply CORS to them. To refuse them, check `Origin` on the upstream
> server.

## What UNCORS changes in proxied traffic

UNCORS forwards the path, query, and body as they came, and changes the
following:

- `Origin` and `Referer` are rewritten from the local host to the upstream
  host. Values that name another host, and `Origin: null`, are forwarded
  unchanged.
- `Location` in redirects is rewritten back to the local host.
- The `Domain` of cookies set by the upstream server becomes the local host
  name. `Secure` follows the local scheme, and on an `http` local host
  `SameSite=None` becomes `SameSite=Lax`, because browsers reject
  `SameSite=None` without `Secure`.
- CORS headers are added to every response, replacing any the upstream server
  sent (unless the origin is outside [`allowed-origins`](#allowed-origins)):

| Header                             | Value                                                                    |
| ---------------------------------- | ------------------------------------------------------------------------ |
| `Access-Control-Allow-Origin`      | The request's `Origin`, or `*`                                           |
| `Access-Control-Allow-Credentials` | `true`                                                                   |
| `Access-Control-Allow-Methods`     | `GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS`                           |
| `Access-Control-Expose-Headers`    | The names of the response headers if the request has an `Origin`, or `*` |
| `Access-Control-Max-Age`           | `86400`                                                                  |
| `Vary`                             | `Origin` is added when the request has an `Origin`                       |

Exposed headers are listed by name because browsers read `*` literally for
requests sent with credentials. Mock and script responses get the same CORS
headers. Files served by [statics](Static-File-Serving) get none.

WebSocket connections are proxied too: use `ws://` with an `http` mapping and
`wss://` with an `https` mapping. They are never cached or recorded to HAR, and
they are closed when UNCORS stops or reloads.

Streaming responses, such as server-sent events and responses without
`Content-Length`, are passed on as each chunk arrives.

<details>
<summary>Full list of changes</summary>

For requests forwarded to the upstream server, UNCORS:

- passes the path and query on as they came: repeated slashes, `.` and `..`
  segments, and escaped characters such as `%2F` are not cleaned or decoded;
- rewrites the `Origin` and `Referer` request headers from the local host to the
  upstream host. An `Origin` or `Referer` of any other host (for example a
  frontend served from `127.0.0.1` or another local domain), and
  `Origin: null`, is forwarded unchanged;
- forwards the request body with its `Content-Length`;
- drops hop-by-hop headers in both directions: `Connection` and the headers it
  names, `Keep-Alive`, `Proxy-Connection`, `Proxy-Authorization`,
  `Proxy-Authenticate`, `TE` (except `TE: trailers`), `Trailer`,
  `Transfer-Encoding`, and `Upgrade` (kept for upgrade requests);
- keeps the upstream `Content-Length` of `HEAD` and `304 Not Modified`
  responses;
- waits up to 5 minutes for the upstream server to start its response (30
  seconds to connect, 10 seconds for the TLS handshake). The response body has
  no time limit: it is forwarded until the upstream server finishes it or the
  client disconnects;
- passes event streams (`text/event-stream`) and responses without a
  `Content-Length` on to the client as each chunk arrives, so server-sent events
  and long downloads are not held back;
- forwards the `Cookie` request header unchanged, and rewrites the `Domain` of
  cookies set by the upstream server to the local host name, without the port.
  A `Domain` that names another host, such as the parent domain of the upstream
  server, is removed, so the cookie belongs to the local host. The `Secure` flag
  of those cookies follows the scheme of the local host. On an `http` local
  host, `SameSite=None` becomes `SameSite=Lax`, because browsers reject a
  `SameSite=None` cookie without `Secure`;
- rewrites `Location` headers in redirects back to the local host;
- proxies WebSocket connections and other requests that ask to switch
  protocols (`Connection: Upgrade` with an `Upgrade` header). The handshake is
  forwarded with its headers, `Origin` and `Referer` rewritten as above. When
  the upstream server answers `101 Switching Protocols`, UNCORS passes that
  response and its headers on to the client and then copies data both ways
  until either side closes the connection. Any other answer is returned as a
  regular response. Use `ws://` with an `http` mapping and `wss://` with an
  `https` mapping. These connections are never cached or recorded to HAR, and
  they are closed when UNCORS stops or reloads its configuration. A mock or a
  script whose route matches the handshake answers it like any other request;
- adds the CORS headers listed below to every response, replacing any the
  upstream server sent, unless the origin is not in
  [`allowed-origins`](#allowed-origins). `Access-Control-Allow-Headers` from
  the upstream server is removed: it only matters in preflight responses.

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

</details>

> [!NOTE]
> UNCORS waits up to 5 minutes for the upstream server to start a response (30
> seconds to connect, 10 seconds for the TLS handshake). The body itself has no
> time limit. Hop-by-hop headers such as `Connection` and `Transfer-Encoding`
> are not forwarded.

## HTTPS configuration

UNCORS signs a certificate for each `https://` host on the fly with a local
certificate authority (CA) that you create once:

```bash
uncors generate-certs                      # valid for 365 days
uncors generate-certs --validity-days 730  # another validity period
uncors generate-certs --force              # replace an existing CA
```

This writes two files to `~/.config/uncors/`:

- `ca.crt`, the CA certificate, which you add to your system's trust store;
- `ca.key`, its private key, readable only by you (`0600`).

Without `--force`, the command stops if either file already exists.
`--validity-days` must be a positive number.

Trust `ca.crt` so browsers accept the certificates:

- macOS: open `ca.crt`, add it to Keychain Access, and set it to Always Trust.
- Linux: copy it to `/usr/local/share/ca-certificates/` and run
  `sudo update-ca-certificates`.
- Windows: import it into Trusted Root Certification Authorities with
  `certmgr.msc`.

Firefox has its own certificate store; see
[Troubleshooting](Troubleshooting#https-certificate-errors).

Then use `https://` in `from`:

```yaml
mappings:
  - from: https://localhost:8443
    to: https://github.com
```

UNCORS refuses to start if an `https://` mapping exists and the CA files are
missing. The HTTPS listener starts only when at least one mapping uses an
`https://` source.

UNCORS signs a certificate only for the host names of its `https://` mappings,
picked by the name the client sends (SNI), and keeps it in memory. A `{name}`
placeholder matches one label, as in routing: `https://{tenant}.local.dev`
covers `acme.local.dev` but not `a.b.local.dev`. For any other host name the
TLS handshake fails, and UNCORS shows a notice once per host. After a reload,
the new host names take effect.

Host certificates use ECDSA P-256 keys and are valid for a year, but never
past the expiry of the CA.

### CA expiry and renewal

Within a week of the CA expiring, UNCORS warns but keeps serving HTTPS. Once it
has expired, HTTPS stops working. To renew:

```bash
uncors generate-certs --force
```

Then trust the new `ca.crt` and restart UNCORS: a running UNCORS keeps signing
with the CA it loaded at startup.

> [!NOTE]
> A client that connects to an IP address sends no host name, so it only works
> with a mapping such as `https://127.0.0.1:8443`. Connect by host name
> otherwise.

## Proxy configuration

UNCORS sends upstream requests through the proxy from `HTTP_PROXY`,
`HTTPS_PROXY`, and `NO_PROXY` (or their lowercase forms). Their values can be
a full URL or just `host:port`, which is treated as HTTP. The `proxy` option or
the `--proxy` flag overrides them and must be a full URL with a scheme:

```yaml
proxy: http://proxy.example.com:8080
```

```bash
uncors --proxy http://proxy.example.com:8080 --from http://localhost --to https://api.example.com
```

For a proxy with authentication, put the credentials in the URL:
`http://username:password@proxy.example.com:8080`.

## Tracing and diagnostics

The console shows handled requests and problems you need to act on, such as a
HAR file that cannot be written. Internal diagnostics are off by default.

Set `UNCORS_TRACE=true` to export [OpenTelemetry](https://opentelemetry.io/)
traces, logs, and metrics over OTLP/HTTP. The endpoint defaults to
`http://localhost:4318`; set `OTEL_EXPORTER_OTLP_ENDPOINT` to send elsewhere,
or its `_TRACES_`, `_LOGS_`, and `_METRICS_` variants to split the signals.
Headers, timeouts, and TLS use the other standard `OTEL_EXPORTER_OTLP_*`
variables. Metrics are exported every 10 seconds unless
`OTEL_METRIC_EXPORT_INTERVAL` (in milliseconds) says otherwise.
`OTEL_LOGS_EXPORTER=none` and `OTEL_METRICS_EXPORTER=none` turn a signal off.

To see everything locally, run [otelop](https://github.com/mashiro/otelop) and
open <http://localhost:4319>:

```bash
docker run --rm -p 4318:4318 -p 4319:4319 ghcr.io/mashiro/otelop
UNCORS_TRACE=true uncors
```

Jaeger works for traces only:

```bash
docker run --rm -p 16686:16686 -p 4318:4318 jaegertracing/jaeger
UNCORS_TRACE=true OTEL_LOGS_EXPORTER=none OTEL_METRICS_EXPORTER=none uncors
```

Without the two `none` settings UNCORS warns that Jaeger rejects logs and
metrics.

Each request is one trace, with child spans for the handlers it passed through.
While tracing is on, every response carries an `X-Uncors-Trace-Id` header, so a
request from the browser's developer tools can be found in the trace viewer,
and the terminal UI shows a `TRACING` badge. UNCORS shows one warning for each
signal it cannot export, for example when no collector is listening, and waits
up to 5 seconds for the last data when it exits.

Failures you need to act on are shown in the console whether tracing is on or
not: a HAR file that cannot be written, a config watcher error, a TLS
certificate that cannot be loaded or generated, a TLS request for a host no
`https://` mapping covers, and requests cut off by a restart.

> [!NOTE]
> Credentials and cookies are never recorded, and no trace context is added to
> proxied requests. `UNCORS_TRACE_CONNECTIONS=true` also records the DNS,
> connect, and TLS phases of upstream requests; it is off by default because
> it nearly doubles the cost of tracing.

<details>
<summary>Spans, logs, and metrics</summary>

#### Traces

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
`supervisor.reload`, `config.load`, `server.listen`, `server.close_in_flight`,
`tls.get_certificate`
(with the SNI name and ALPN protocols), `tls.ensure_ca` (with the CA expiry),
`tls.generate_certificate`, `har.flush`, and `version.check`.

#### Logs

Logs are exported with the trace and span they belong to, so a trace viewer
shows them next to the request that produced them.

| Level   | Logged                                                                                                                                                                                            |
| ------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ERROR` | A request a handler failed (method, URL, error); a HAR file that cannot be written.                                                                                                               |
| `WARN`  | An upstream `5xx` response; a Lua script error, with its stack trace; a HAR entry dropped because the write queue is full.                                                                        |
| `DEBUG` | Each routing decision: the upstream URL and status, cache hits, misses, and stores, answered preflights, rewrites, static files served or passed on, mock responses, script runs, and HAR writes. |

A request the client cancels is logged at `DEBUG`, not as a failure.

#### Metrics

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

</details>
