# CLAUDE.md

UNCORS is a development-only HTTP/HTTPS proxy that rewrites CORS headers, mocks
responses, runs Lua scripts, serves static files, caches responses, rewrites
requests, and records HAR files. Details: `ARCHITECTURE.md`, `CONTRIBUTING.md`.

## Commands

```bash
make check              # format + test + build (build only compiles packages)
make test               # Unit tests with -race
make test-integration   # tests/integration, -tags integration (real sockets + TLS)
make test-release       # Unit tests with -tags release
make build-release      # Build the ./uncors binary
make format-docs        # Prettier over docs/*.md and root *.md
./uncors generate-certs # Local CA in ~/.config/uncors (needed for https mappings)
```

`make format-docs` also realigns tables in untouched files; revert those before
committing.

## Architecture

Request flow:

1. `internal/server`: one listener per port/scheme group of mappings.
2. `internal/handler/router`: picks the mapping by host, then the first matching
   route: statics → mocks → scripts → rewrites → default handler.
3. Default handler (`prepareDefaultHandler`): HAR → cache → OPTIONS → proxy.

Statics and rewrites fall through to the default handler, so only proxied
traffic is cached and recorded to HAR. Proxy, mock, and script handlers add CORS
headers; statics do not.

Core and UI are separate (`TestCoreDoesNotImportTheUI` enforces it):

- `internal/engine`: runs the proxy headless. `Engine` serves one `Generation`
  (everything built from one `UncorsConfig`) and swaps it on reload; Start,
  Restart, Shutdown, and Close are serialised. Register anything
  config-derived that holds resources on the `Generation` so a reload releases
  it.
- `internal/events`: the engine reports only by publishing events on a `Bus`
  (non-blocking; slow subscribers drop). Each request publishes exactly
  `RequestStarted` and `RequestCompleted`; the handler that answers calls
  `tracking.Claim` (`internal/tracking`).
- `internal/cli` supervisor: owns the engine, the config watcher, signals,
  and the version check; frontends send it `events.Restart` / `events.Shutdown`.
  An invalid file publishes `ReloadFailed` and keeps the previous config.
- Frontends only render events: the TUI (`internal/uncors_app`, BubbleTea, the
  default) and the headless printer (`--interactive=false`), both through
  `tui.EventRenderer`.

## Conventions

- Handlers implement `contracts.Handler`, middleware `contracts.Middleware`;
  build them in `engine.Generation` and wire them in `internal/handler/router`.
  A handler that produces a response calls `tracking.Claim`.
- Constructors use functional options (`helpers.ApplyOptions`).
- Use the injected `afero.Fs`, never `os`.
- YAML tags are kebab-case.
- Mocks: minimock into `testing/mocks`. Snapshots: go-snaps. Helpers:
  `testing/testutils`.
- Commit prefixes: `feat:`, `fix:`, `docs:`, `refactor:`, `test:`.

## Configuration

- `internal/config` uses pflag + yaml.v3; flags override the file. Validation is
  plain Go.
- `schema.json` is editor/test-only (not used at runtime); keep it in sync by
  hand.
- A new option needs: struct field, validation, `schema.json`, a fixture in
  `tests/schema/`, and docs in `docs/`.
- `--interactive` is a CLI flag only, not a YAML key.

## Documentation

`docs/*.md` is published to the GitHub wiki on release, so it must match main.
Update the relevant page when behavior changes; put breaking changes since the
last release in `docs/Migration-Guide.md`.

## Debugging

`UNCORS_LOGGING=/path/to/file` appends the internal log to that file; otherwise
it is discarded. Log through `log/slog` (`internal/logging`); components that
report errors take a `*slog.Logger` option. Never write diagnostics to the
terminal: only the UI/printer renders there.
