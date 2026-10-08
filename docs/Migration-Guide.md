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

### `debug` is replaced by `UNCORS_LOGGING`

The `--debug` flag and the `debug` configuration key are gone. `--debug` now
fails with an unknown-flag error, and a `debug` key in the file is ignored. To
get diagnostic logs, set `UNCORS_LOGGING` to a file path:

```bash
# 0.6.x
uncors --debug --config .uncors.yaml

# Now
UNCORS_LOGGING=uncors.log uncors --config .uncors.yaml
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
3. Run with diagnostic logging:
   `UNCORS_LOGGING=uncors.log uncors --config .uncors.yaml`.
4. Report the problem on [GitHub Issues](https://github.com/evg4b/uncors/issues).
