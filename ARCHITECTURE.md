# UNCORS architecture

How UNCORS handles a request and where the code for each part lives.

## Overview

UNCORS is a local reverse proxy for development. It listens on the ports named
in the `from` URLs of its mappings, picks a mapping by the request's host name,
and either answers the request itself (static file, mock, script) or forwards
it to the mapping's `to` URL. Responses get permissive CORS headers so the
browser accepts them.

## Packages

| Package                 | Responsibility                                                                                     |
| ----------------------- | -------------------------------------------------------------------------------------------------- |
| `main.go`               | Builds the DI container and dispatches to `generate-certs` or the main command.                    |
| `internal/cli`          | Command entry points: `RunUncors` (interactive or headless mode, config watching, shutdown) and `GenerateCerts`. |
| `internal/commands`     | The `generate-certs` command implementation.                                                       |
| `internal/di`           | `Container` (process-lifetime services), `Runtime` (everything built from one config), `Proxy` (serves a `Runtime` and swaps it on reload). |
| `internal/config`       | Flag parsing, YAML loading, normalisation, validation, and the file watcher.                       |
| `internal/handler`      | Router, handlers, and middleware (see below).                                                      |
| `internal/server`       | Listeners, TLS with on-the-fly host certificates, the local CA, and request tracking.              |
| `internal/infra`        | HTTP client, CORS header helpers, error pages, diagnostic logging setup.                           |
| `internal/urlreplacer`  | Matching `from` hosts (with `{name}` placeholders) and building upstream URLs.                     |
| `internal/tui`, `internal/uncors_app` | Console output and the interactive Bubble Tea UI.                                    |
| `internal/version`      | Checks GitHub for a newer release at startup.                                                      |
| `pkg/urlt`              | A fork of `net/url` with placeholder support.                                                      |
| `testing/`, `tests/`    | Shared test helpers and mocks; integration and schema tests.                                       |

## Configuration

`config.LoadConfiguration` parses flags with `pflag`, decodes the YAML file
with `gopkg.in/yaml.v3`, applies flag overrides, normalises mappings, and
validates the result in Go code. Validation also checks that referenced files
and directories exist and that the local CA exists when an HTTPS mapping is
configured.

`schema.json` describes the same format for editors and is checked by the tests
in `tests/schema`. It is not used at runtime, so changes to the config structs
must be mirrored there by hand.

`config.Watcher` watches the configuration file. On a change, the CLI reloads
the configuration and calls `di.Proxy.Restart`, which builds a new `Runtime`
before replacing the old one. If loading fails, the old configuration stays
active.

## Request flow

Each listener serves the mappings that share its port and scheme. A
`gorilla/mux` router (`internal/handler/router`) selects the mapping by host
name and then the first matching route, in this order:

1. Static directories (`statics`), matched by path prefix.
2. Mocks, then scripts. Within each, entries with a method, query, or header
   filter come before entries that match on path alone.
3. Rewrites, matched by path or path prefix.
4. The mapping's default handler, for everything else.

Requests with a host that no mapping covers get a "host not mapped" error.

The default handler is a chain of middleware around the proxy handler:

```
HAR collector → cache → OPTIONS handling → proxy
```

- The HAR collector (if `har` is set) records the request and response.
- The cache (if `cache` globs are set) answers repeated requests from memory
  and stores 2xx responses.
- OPTIONS handling (unless disabled) answers preflight requests.
- The proxy rewrites the URL, `Origin`, `Referer`, cookies, and `Location`,
  forwards the request, and adds CORS headers to the response.

Static directories and rewrites hand requests they don't serve themselves to
this same default handler, so those requests are cached and recorded like
any other proxied request. Mock and script responses don't pass through it:
they are not cached or recorded, and they add CORS headers themselves. Files
served from static directories get no CORS headers.

## Middleware and handler contracts

The contracts live in `internal/contracts`:

```go
type Handler interface {
	ServeHTTP(writer ResponseWriter, request *Request) error
}

type Next func(writer ResponseWriter, request *Request) error

type Middleware interface {
	ServeHTTP(writer ResponseWriter, request *Request, next Next) error
}
```

Handlers return errors instead of writing error pages themselves; the
infrastructure layer turns them into responses. Components are built with
functional options (`helpers.ApplyOptions`) and wired in `internal/di`.

## HAR collector

`internal/handler/har` records traffic in HAR 1.2 format.

- Requests never wait for disk I/O. The middleware sends each entry over a
  channel with capacity 4096 and drops the entry if the channel is full.
- One background goroutine per mapping drains the channel and rewrites the
  whole file after each batch, writing a temporary file and renaming it over
  the target, so the file on disk is always valid.
- Each `Writer` belongs to the `Runtime` that created it. When the runtime is
  closed on shutdown or reload, `Close` drains the channel and writes the file
  one last time. A new runtime starts a new, empty recording.
- `Cookie`, `Set-Cookie`, `Authorization`, `WWW-Authenticate`,
  `Proxy-Authorization`, and `Proxy-Authenticate` are left out unless
  `capture-secure-headers` is set.

## HTTPS

`uncors generate-certs` creates a CA in `~/.config/uncors/`. At runtime,
`server.HostCertManager` loads it on the first TLS handshake, signs a
certificate for the SNI host name, and caches it in memory.

## Extending UNCORS

A new request handler:

1. Add a package under `internal/handler/`.
2. Implement `contracts.Handler`.
3. Add a constructor to the DI layer and register routes for it in
   `internal/handler/router`.
4. Add the config struct, validation, and `schema.json` entries.

New middleware:

1. Add a package under `internal/handler/`.
2. Implement `contracts.Middleware`.
3. Wire it into the router, usually in `prepareDefaultHandler`.

A new config option:

1. Add the field to the structs in `internal/config/`, with validation.
2. Add it to `schema.json` and a fixture under `tests/schema/`.
3. Document it in `docs/`.

## Testing

- Unit tests sit next to the code and use mocks generated with `minimock`.
- `tests/integration` holds end-to-end tests with real sockets and TLS; run
  them with `make test-integration`.
- `tests/schema` validates YAML fixtures against `schema.json`.
- `make test` runs the unit tests with the race detector.

UNCORS is for local development only. Don't expose it to the internet.
