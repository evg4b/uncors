This page lists breaking changes between UNCORS versions and shows how to
update your configuration.

## Contents

- [Upgrading from 0.6.x](#upgrading-from-06x)
- [Upgrading from 0.5.x to 0.6.x](#upgrading-from-05x-to-06x)
  - [TLS certificates](#tls-certificates)
  - [Ports](#ports)
  - [Fake data responses](#fake-data-responses)

## Upgrading from 0.6.x

These changes are on the main branch and will ship in the next release after
0.6.1.

### Placeholders replace the `*` wildcard

Host wildcards are now written as named placeholders. A `*` in `from` or `to`
is a configuration error. Give each wildcard a name, and use the same name in
the target to carry the value over:

In 0.6.x:

```yaml
mappings:
  - from: http://*.local.com:8080
    to: https://*.example.com
```

Now:

```yaml
mappings:
  - from: http://{sub}.local.com:8080
    to: https://{sub}.example.com
```

With several placeholders, values are matched by name rather than by position.
See [Named placeholder mapping](Configuration#named-placeholder-mapping).

### `debug` is replaced by `UNCORS_TRACE`

The `--debug` flag and the `debug` configuration key are gone. `--debug` now
fails with an unknown-flag error, and a `debug` key in the file is ignored. To
get diagnostics, set `UNCORS_TRACE=true` and run an OpenTelemetry collector
such as Jaeger. UNCORS exports traces and logs to it over OTLP/HTTP
(`localhost:4318` by default). See
[Tracing and diagnostics](Configuration#tracing-and-diagnostics).

```bash
# 0.6.x
uncors --debug --config .uncors.yaml

# Now
UNCORS_TRACE=true uncors --config .uncors.yaml
```

### Cache settings

`cache-config.clear-time` is gone and is ignored if present. A new
`cache-config.max-size` limits the total size of the cache in bytes (100 MB by
default):

In 0.6.x:

```yaml
cache-config:
  expiration-time: 10m
  clear-time: 5m
```

Now:

```yaml
cache-config:
  expiration-time: 10m
  max-size: 52428800
```

### Durations without spaces

`delay` and `expiration-time` are parsed with Go's duration syntax, which does
not allow spaces. Values like `1m 30s` now fail with a parse error:

In 0.6.x:

```yaml
delay: 1m 30s
```

Now:

```yaml
delay: 1m30s
```

### Terminal UI by default

UNCORS now starts an interactive terminal UI. Pass `--interactive=false` to
get plain log output, for example in scripts, CI, or Docker.

### Request log labels and errors

Each request line is labelled with what produced the response: `PROXY`,
`CACHE`, `MOCK`, `SCRIPT`, `STATIC`, or `OPTIONS`. A cache hit is now labelled
`CACHE` and a cache miss `PROXY`. Rewrites are no longer a label; the rewritten
path follows the URL after `→`. See
[Reading the request log](Troubleshooting#reading-the-request-log).

Handler errors are no longer printed as separate console lines. They appear
under the failing request instead, and are also recorded on the request's trace
when [tracing](Configuration#tracing-and-diagnostics) is on.

### Request paths are no longer cleaned

UNCORS used to answer a path with repeated slashes or `.` and `..` segments,
such as `//api//v1`, with a `301` redirect to the cleaned path. The redirect
skipped the mapping's mocks, scripts, and proxy, carried no CORS headers, and
made browsers repeat a `POST` as a `GET`. Such paths are now handled as they
came: they are proxied to the upstream unchanged, and mocks, scripts, statics,
and rewrites match them only if their `path` matches the path as written. If a
client sends such paths to reach a mock, fix the client or add the path to the
mock.

### Script limits and `os.exit`

Lua scripts now have limits. See [Limits](Script-Handler#limits).

- `os.exit` raises an error instead of stopping the UNCORS process. A script
  that called it to end early should `return` from the top level instead.
- A script that runs longer than 30 seconds is stopped, and the request fails
  with status 500. Set `timeout` on a script that needs longer. A script whose
  client disconnects or whose server shuts down is stopped too, and the
  connection is closed.
- A request body larger than 10 MiB fails with status 500 before the script
  runs. Set `max-body-size` on a script that needs larger bodies. Before, a
  body that failed to read reached the script as `nil`.

### One HAR file per mapping

Two mappings can no longer record to the same [HAR](HAR-Collector) file.
UNCORS compares the absolute paths, so `./api.har` and `recordings/../api.har`
count as the same file. Such a configuration fails to load with an error like:

```text
mappings[1].har: HAR file "./api.har" is already used by mappings[0].har
```

Before, both mappings rewrote the same file, and one of them overwrote the
entries of the other or failed to write it. Give each mapping its own file.

### Error page status codes

UNCORS error pages used to have status 500 for every failure. The status now
depends on the cause:

- `404` when no mapping matches the host.
- `502` when the upstream server cannot be reached.
- `504` when the upstream request times out.
- `500` for other failures, as before.

Error pages now carry CORS headers, so the browser shows the error instead of a
CORS error, and they no longer include memory usage. When the response has
already started, or the client cancelled the request, UNCORS closes the
connection instead of writing an error page. Code that checks for status 500
from UNCORS should accept the new codes. See
[Error pages](Troubleshooting#error-pages).

### HTTPS certificates only for mapped hosts

UNCORS used to sign a certificate for any host name a client asked for. It now
signs certificates only for the host names of the `https://` mappings, and the
TLS handshake fails for any other name with the notice
"TLS: no https mapping for host". A `{name}` placeholder matches one label of
the host name. A client that connects to an IP address without a host name
gets a certificate only when a mapping uses that address, such as
`https://127.0.0.1:8443`. If a client reaches UNCORS under a name that is not
in `from`, add a mapping for it. See
[Use HTTPS mappings](Configuration#use-https-mappings).

Host certificates now use ECDSA P-256 keys instead of RSA, and end no later
than the CA does. Existing CAs keep working.

`uncors generate-certs --validity-days` with zero or a negative number is now
an error instead of creating an expired CA.

### Status codes of file mocks

A mock with a `file` response used to be sent with status 200 whatever its
`code` was. It now has the status set in `code`. A mock that has a `code` other
than 200 and relied on getting 200 should set `code: 200`. Range and
conditional requests are answered only for mocks with `code: 200`. See
[Body from `file`](Response-Mocking#body-from-file).

## Upgrading from 0.5.x to 0.6.x

Version 0.6 had three breaking changes.

### TLS certificates

The top-level `cert-file` and `key-file` options were removed. Instead of one
certificate shared by every HTTPS mapping, UNCORS now creates a certificate
for each host, signed by a local CA.

In 0.5.x:

```yaml
cert-file: ~/certs/server.crt
key-file: ~/certs/server.key
mappings:
  - from: https://app.local:8443
    to: https://api.example.com
```

In 0.6.x:

```yaml
mappings:
  - from: https://app.local:8443
    to: https://api.example.com
```

To migrate:

1. Remove `cert-file` and `key-file`.
2. Run `uncors generate-certs` once. It creates `~/.config/uncors/ca.crt` and
   `~/.config/uncors/ca.key`.
3. Add `ca.crt` to your system's trusted certificates so browsers accept the
   generated certificates.

Several HTTPS hosts can now share a port, because the certificate is picked by
SNI.

### Ports

The top-level `http-port` and `https-port` options and the matching flags were
removed. Each mapping now sets its own port in the `from` URL. Without a port,
UNCORS uses 80 for HTTP and 443 for HTTPS.

In 0.5.x:

```yaml
http-port: 8080
https-port: 8443
mappings:
  - from: http://localhost
    to: https://api.example.com
  - from: https://secure-app
    to: https://backend.example.com
  - http://other: https://github.com
```

In 0.6.x:

```yaml
mappings:
  - from: http://localhost:8080
    to: https://api.example.com
  - from: https://secure-app:8443
    to: https://backend.example.com
  - http://other:8080: https://github.com
```

On the command line:

```bash
# 0.5.x
uncors --http-port 8080 --from http://localhost --to https://api.example.com

# 0.6.x
uncors --from http://localhost:8080 --to https://api.example.com
```

Mappings can now listen on different ports:

```yaml
mappings:
  - from: http://api.local:3000
    to: https://api.example.com
  - from: http://admin.local:4000
    to: https://admin.example.com
  - from: https://secure.local:8443
    to: https://backend.example.com
```

### Fake data responses

The `fake` response type for mocks was removed. Generate dynamic data with a
[script](Script-Handler) instead. A script can call an external generator such
as [fakedata](https://github.com/lucapette/fakedata):

In 0.5.x:

```yaml
mappings:
  - from: http://localhost:8080
    to: https://api.example.com
    mocks:
      - path: /api/users
        response:
          code: 200
          fake:
            type: object
            properties:
              login:
                type: email
              username:
                type: username
          seed: 12345
```

In 0.6.x:

```yaml
mappings:
  - from: http://localhost:8080
    to: https://api.example.com
    scripts:
      - path: /api/users
        script: |
          local handle = io.popen("fakedata --format=ndjson --limit 1 login=email username=username")
          local output = handle:read("*a")
          handle:close()

          response.headers["Content-Type"] = "application/json"
          response:WriteHeader(200)
          response:WriteString(output)
```

For an array, collect the NDJSON lines into a JSON array:

```yaml
scripts:
  - path: /api/users
    script: |
      local handle = io.popen("fakedata --format=ndjson --limit 5 name=name email=email")
      local output = handle:read("*a")
      handle:close()

      local lines = {}
      for line in output:gmatch("[^\n]+") do
        table.insert(lines, line)
      end

      response.headers["Content-Type"] = "application/json"
      response:WriteHeader(200)
      response:WriteString("[" .. table.concat(lines, ",") .. "]")
```

Install fakedata with Homebrew (`brew install lucapette/tap/fakedata`) or Go
(`go install github.com/lucapette/fakedata@latest`). `io.popen` runs the
command on every request, with the permissions of the UNCORS process.

## Getting help

If a migration doesn't work:

1. Compare your file with the [Configuration](Configuration) reference.
2. Validate it against the
   [JSON Schema](https://raw.githubusercontent.com/evg4b/uncors/main/schema.json).
3. Run with [tracing](Configuration#tracing-and-diagnostics):
   `UNCORS_TRACE=true uncors --config .uncors.yaml`.
4. Report the problem on [GitHub Issues](https://github.com/evg4b/uncors/issues).
