You can configure UNCORS with command-line flags, a YAML file, or both. When
you use both, flags take precedence over the file.

Everything revolves around host mappings. A mapping says "requests that arrive
at this local address go to that remote address". Global options such as the
upstream proxy apply to all mappings. Per-mapping options such as mocks or
static files apply only to the mapping they are defined in.

## Contents

- [Quick reference](#quick-reference)
- [Command-line flags](#command-line-flags)
- [Diagnostic logging](#diagnostic-logging)
- [Configuration file](#configuration-file)
- [Global options](#global-options)
- [Mapping options](#mapping-options)
  - [Ports](#ports)
  - [Protocol scheme mapping](#protocol-scheme-mapping)
  - [Named placeholder mapping](#named-placeholder-mapping)
  - [Simplified syntax](#simplified-syntax)
  - [OPTIONS request handling](#options-request-handling)
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
new mapping.

```bash
uncors --from http://localhost:8080 --to https://api.example.com \
       --from http://localhost:8081 --to https://auth.example.com
```

To generate the local CA used for HTTPS mappings, run
`uncors generate-certs`. See [HTTPS configuration](#https-configuration).

## Diagnostic logging

UNCORS always prints handled requests to the console. Internal diagnostic logs
are off by default. To capture them, set the `UNCORS_LOGGING` environment
variable to a file path:

```bash
UNCORS_LOGGING=uncors.log uncors --config .uncors.yaml
```

Logs are appended to the file. If the variable is unset, empty, or points to a
path that cannot be opened, diagnostic output is discarded.

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
`from: to` pair:

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

| Header                             | Value                                                                     |
| ---------------------------------- | ------------------------------------------------------------------------- |
| `Access-Control-Allow-Origin`      | The request's `Origin`, or `*` if there is none                           |
| `Access-Control-Allow-Methods`     | The request's `Access-Control-Request-Method`, or a list of common methods |
| `Access-Control-Allow-Headers`     | The request's `Access-Control-Request-Headers`, or `*`                    |
| `Access-Control-Allow-Credentials` | `true`                                                                    |
| `Access-Control-Expose-Headers`    | `*`                                                                       |
| `Access-Control-Max-Age`           | `86400`                                                                   |

| Option     | Type    | Default | Description                                                    |
| ---------- | ------- | ------- | -------------------------------------------------------------- |
| `disabled` | boolean | `false` | Forward `OPTIONS` requests to the upstream server instead.     |
| `code`     | integer | `200`   | Status code of the `OPTIONS` response.                         |
| `headers`  | object  | -       | Extra headers. They replace the defaults when names collide.   |

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

## What UNCORS changes in proxied traffic

For requests forwarded to the upstream server, UNCORS:

- rewrites the `Origin` and `Referer` request headers from the local host to the
  upstream host;
- forwards cookies, and rewrites the `Domain` of cookies set by the upstream
  server to the local host. The `Secure` flag follows the scheme of the side
  the cookie is sent to;
- rewrites `Location` headers in redirects back to the local host;
- adds the CORS headers listed below to every response, replacing any the
  upstream server sent.

| Header                             | Value                                                                    |
| ---------------------------------- | ------------------------------------------------------------------------ |
| `Access-Control-Allow-Origin`      | The request's `Origin`, or `*` if there is none                          |
| `Access-Control-Allow-Credentials` | `true`                                                                   |
| `Access-Control-Allow-Headers`     | `*`                                                                      |
| `Access-Control-Allow-Methods`     | `GET, PUT, POST, HEAD, TRACE, DELETE, PATCH, COPY, HEAD, LINK, OPTIONS`  |
| `Access-Control-Expose-Headers`    | `*`                                                                      |
| `Access-Control-Max-Age`           | `86400`                                                                  |

Mock and script responses get the same CORS headers. Files served by
[statics](Static-File-Serving) do not.

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
- `ca.key`, its private key.

Without `--force`, the command stops if either file already exists.

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
are missing. It creates a certificate for each host name the client asks for
(through SNI) and keeps it in memory, so clients must connect by host name, not
by IP address. The HTTPS listener only starts when at least one mapping uses an
`https://` source.

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
