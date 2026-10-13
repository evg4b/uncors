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
| `main.go`               | Sets up telemetry, dispatches to `generate-certs` or the main command, and flushes it on exit.     |
| `internal/cli`          | Command entry points (`RunUncors`, `GenerateCerts`), the supervisor that owns the engine, and the two frontends. |
| `internal/commands`     | The `generate-certs` command implementation.                                                       |
| `internal/engine`       | Runs the proxy without UI: `Engine` (serves a `Generation` and swaps it on reload), `Generation` (everything built from one config). |
| `internal/config`       | Flag parsing, YAML loading, normalisation, validation, and the file watcher.                       |
| `internal/handler`      | Router, handlers, and middleware (see below).                                                      |
| `internal/server`       | Listeners, TLS with on-the-fly host certificates, and the local CA.                               |
| `internal/infra`        | HTTP client, CORS header helpers, error pages.                                                     |
| `internal/events`       | Events the engine publishes, commands frontends send, and the non-blocking `Bus`.                |
| `internal/tracking`     | Per-request scope: attributes each request to the component that answered it and publishes its two events. |
| `internal/telemetry`    | OpenTelemetry spans and logs, exported over OTLP when `UNCORS_TRACE=true`; free when off.         |
| `internal/urlreplacer`  | Matching `from` hosts (with `{name}` placeholders) and building upstream URLs.                     |
| `internal/tui`, `internal/uncors_app` | Event rendering, console output, and the interactive Bubble Tea UI.                  |
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

`config.Watcher` watches the configuration file. On a change, the supervisor
reloads the configuration and calls `engine.Restart`, which builds a new
`Generation` before replacing the old one. If loading fails, the supervisor
publishes `ReloadFailed` and the old configuration stays active. `Engine` runs
one start, restart, or shutdown at a time, and a restart closes requests that
are still running after the shutdown timeout, so the new listeners always
start.

## Core, supervisor, and frontends

```
cli.RunUncors ─► supervisor ──Start/Restart/Shutdown──► engine.Engine ──► server, router, handlers
                    ▲   │                                      │
   Restart/Shutdown │   │ ServerStopped, ReloadFailed,         │ ServerStarted/Restarted,
                    │   ▼ NewVersionAvailable                  ▼ RequestStarted/Completed
                 frontend ◄──────────────── events.Bus ◄───────┘
         (uncors_app TUI or headless printer)
```

- The engine (`internal/engine`, plus `server`, `handler`, `events`,
  `tracking`) has no UI dependency; `TestCoreDoesNotImportTheUI` enforces it.
  It reports only by publishing events, so tests can run it with a bus
  subscription and no frontend.
- The supervisor (`internal/cli`) is the only goroutine that starts, restarts,
  and stops the engine. It reacts to frontend commands (`events.Restart`,
  `events.Shutdown`), config file changes, OS signals, and context
  cancellation, and runs the version check.
- A frontend runs on its own goroutine and only renders events
  (`tui.EventRenderer`) and sends commands. It subscribes before the engine
  starts. `ServerStopped` is the last event; the bus closes after it. If the
  frontend exits on its own, the supervisor shuts down.
- Diagnostics are OpenTelemetry spans and logs (`internal/telemetry`), never
  terminal output. Each request is one trace: `server.handleRequest` starts the
  root span and every handler and middleware adds a child span. Failures the
  user must act on (HAR writes, the config watcher, TLS certificates, cut-off
  requests) are also published as `events.Notice`.

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

## Request events

Every request produces exactly two events on the bus: `RequestStarted` and
`RequestCompleted`. The server opens a `tracking.Scope` per request and puts it
in the request context. A component calls `tracking.Claim` at the moment it
decides to produce the response; the first claim publishes `RequestStarted`,
and `RequestCompleted` reports the last one. Only the component that answers
claims:

- the cache on a hit (a miss is claimed by the proxy that fetches it),
- OPTIONS handling when it answers a preflight,
- a static directory when it serves (or fails to serve) a file,
- mocks, scripts, and the proxy on entry.

Rewrites don't claim; they call `tracking.Rewrote` so both events carry the
path change. HAR only observes. A request nobody claimed (an unmapped host)
gets `HandlerNone`. Handler errors that become error pages are recorded with
`tracking.Fail` and carried in `RequestCompleted.Err`.

`Bus.Publish` never blocks: each subscriber has its own buffer, and events that
don't fit are dropped and counted, so a slow UI can't slow down traffic.

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

Handlers return errors instead of writing error pages themselves;
`infra.CastToHTTPHandler`, which wraps every route, logs them and turns them
into responses. Components are built with
functional options (`helpers.ApplyOptions`) and wired in `engine.Generation`.

## HAR collector

`internal/handler/har` records traffic in HAR 1.2 format.

- Requests never wait for disk I/O. The middleware sends each entry over a
  channel with capacity 4096 and drops the entry if the channel is full.
- One background goroutine per mapping drains the channel and rewrites the
  whole file after each batch, writing a temporary file and renaming it over
  the target, so the file on disk is always valid.
- Each `Writer` belongs to the `Generation` that created it. When the
  generation is closed on shutdown or reload, `Close` drains the channel and
  writes the file one last time. A new generation starts a new, empty
  recording.
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
2. Implement `contracts.Handler`, and call `tracking.Claim` with a new
   `events.HandlerKind` when it produces a response.
3. Add a constructor to `engine.Generation` (and `router.Factory`) and register
   routes for it in `internal/handler/router`.
4. Give the new kind a label in `tui/styles.HandlerLabel`.
5. Add the config struct, validation, and `schema.json` entries.

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
  them with `make test-integration`. The harness (`testing/integration`) loads
  the config through `config.LoadConfiguration` and serves it with an
  `engine.Engine` directly, with no CLI or frontend, and records every bus
  event: `Env.DoTracked` returns a request's `RequestStarted` and
  `RequestCompleted`, `Env.Events` everything published.
  `tests/integration/cli` runs the built binary.
- `tests/schema` validates YAML fixtures against `schema.json`.
- `make test` runs the unit tests with the race detector.

UNCORS is for local development only. Don't expose it to the internet.
