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

Lifetimes (`internal/di`):

- `Container`: process-lifetime services (fs, output, server, request tracker).
- `Runtime`: everything built from one `UncorsConfig`. Register anything
  config-derived that holds resources on it so a reload releases it.
- `Proxy`: serves one `Runtime` and swaps it on reload; Start, Restart,
  Shutdown, and Close are serialised.

Interactive mode (`internal/uncors_app`, BubbleTea) is the default;
`--interactive=false` runs headless (`internal/cli`). Both watch the config file
and call `Proxy.Restart`; an invalid file keeps the previous configuration.

## Conventions

- Handlers implement `contracts.Handler`, middleware `contracts.Middleware`;
  wire new ones in `internal/handler/router` through a DI constructor.
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
