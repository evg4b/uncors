This page lists breaking changes between UNCORS versions and shows how to
update your configuration.

## Contents

- [Upgrading from 0.6.x](#upgrading-from-06x)
  - [Placeholders replace the `*` wildcard](#placeholders-replace-the--wildcard)
  - [`debug` is replaced by `UNCORS_TRACE`](#debug-is-replaced-by-uncors_trace)
  - [Cache settings](#cache-settings)
  - [Durations without spaces](#durations-without-spaces)
  - [Terminal UI by default](#terminal-ui-by-default)
  - [Script limits and `os.exit`](#script-limits-and-osexit)
  - [Unknown keys are errors](#unknown-keys-are-errors)
  - [Simplified mapping syntax needs a host-like key](#simplified-mapping-syntax-needs-a-host-like-key)
  - [Duplicate mappings](#duplicate-mappings)
- [Upgrading from 0.5.x to 0.6.x](#upgrading-from-05x-to-06x)
  - [TLS certificates](#tls-certificates)
  - [Ports](#ports)
  - [Fake data responses](#fake-data-responses)

## Upgrading from 0.6.x

These changes are on the main branch and will ship in the next release after
0.6.1.

### Placeholders replace the `*` wildcard

A `*` in `from` or `to` is now a configuration error. Give each wildcard a
name, and use the same name in the target to carry the value over:

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

With several placeholders, values are matched by name, not by position. See
[Named placeholder mapping](Configuration#named-placeholder-mapping).

### `debug` is replaced by `UNCORS_TRACE`

The `--debug` flag and the `debug` key are gone. `--debug` fails with an
unknown-flag error, and a `debug` key is an
[unknown field](#unknown-keys-are-errors). To get diagnostics, set
`UNCORS_TRACE=true` and run an OpenTelemetry collector such as Jaeger. UNCORS
exports traces and logs to it over OTLP/HTTP (`localhost:4318` by default). See
[Tracing and diagnostics](Configuration#tracing-and-diagnostics).

```bash
# 0.6.x
uncors --debug --config .uncors.yaml

# Now
UNCORS_TRACE=true uncors --config .uncors.yaml
```

### Cache settings

`cache-config.clear-time` is gone and is an
[unknown field](#unknown-keys-are-errors). The cache size is limited by the new
`cache-config.max-size` in bytes (100 MB by default). `CONNECT` is no longer
accepted in `cache-config.methods`; UNCORS never cached it.

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

`delay` and `expiration-time` use Go's duration syntax, which does not allow
spaces. A value like `1m 30s` now fails to parse:

In 0.6.x:

```yaml
delay: 1m 30s
```

Now:

```yaml
delay: 1m30s
```

### Terminal UI by default

UNCORS now starts an interactive terminal UI, which needs a terminal. Without
one, for example in scripts, CI, or Docker, UNCORS fails to start. Pass
`--interactive=false` there to get plain log output.

### Script limits and `os.exit`

Lua scripts now have limits. See [Limits](Script-Handler#limits).

- `os.exit` raises an error instead of stopping the UNCORS process. A script
  that called it to end early should `return` from the top level instead.
- A script that runs longer than 30 seconds is stopped, and the request fails
  with status 500. Set `timeout` on a script that needs longer.
- A request body larger than 10 MiB fails with status 500 before the script
  runs. Set `max-body-size` on a script that needs larger bodies.

### Unknown keys are errors

UNCORS used to ignore keys it did not know, so a typo silently turned an option
off. Now any unknown key fails the configuration, at any level of the file,
with its path and line:

```text
mappings[0].mocks[0].response.cod: unknown field (line 8)
```

Remove keys of removed options, such as `debug`, `cache-config.clear-time`,
`http-port`, `https-port`, `cert-file`, and `key-file`. A top-level key that
only holds a YAML anchor is an unknown key too; move the anchor to a value
inside a known option.

### Simplified mapping syntax needs a host-like key

A mapping with a single `key: value` pair is read as `from: to` only when the
key contains a scheme, a port, or a dot, or is `localhost`. Any other key is an
unknown field. A simplified mapping with a single-label host and no scheme or
port, such as `- myapp: https://example.com`, needs a scheme:

```yaml
mappings:
  - http://myapp: https://example.com
```

### Duplicate mappings

Two mappings with the same `from` are an error. Before, UNCORS used the first
one and ignored the other. `from` is compared after adding the default scheme
and port, ignoring the case of the host name, so `localhost:80` and
`http://LOCALHOST` are the same:

```text
mappings[1].from: http://localhost is already mapped by mappings[0].from
```

Merge the options of the duplicates into one mapping.

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
