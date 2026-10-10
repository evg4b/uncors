# CLAUDE.md

Guidance for Claude Code when working in this repository.

UNCORS is a local development HTTP/HTTPS proxy that rewrites CORS headers. It
also mocks responses, runs Lua scripts, serves static files, caches responses,
rewrites requests, and records HAR files. It is a development-only tool, not
meant for production or remote use. Go version: see `go.mod`.

## Commands

```bash
make build-release      # Build the ./uncors binary (make build only compiles packages)
make test               # Unit tests with -race
make test-integration   # Integration tests in tests/integration (real sockets + TLS)
make test-release       # Tests with the release build tag
make format             # gofmt, gofumpt, golangci-lint --fix (all linters except those disabled)
make check              # format + test + build
make format-docs        # Prettier over docs/*.md and the root *.md files
go test -run TestName ./internal/handler/proxy/
./uncors generate-certs # Create the local CA in ~/.config/uncors (needed for https mappings)
```

`make format-docs` also realigns tables in files you did not touch; revert
unrelated changes before committing.

## Architecture

See `ARCHITECTURE.md` for details.

Request flow:

1. `internal/server`: one listener per port/scheme group of mappings.
2. `internal/handler/router` (gorilla/mux): picks the mapping by host name,
   then the first matching route: statics (path prefix) → mocks → scripts →
   rewrites → the mapping's default handler.
3. Default handler chain: HAR collector → cache → OPTIONS handling → proxy.

Statics and rewrites pass unhandled requests to the default handler, so only
proxied traffic is cached and recorded to HAR; mock, script, and served static
responses are not. CORS headers are added by the proxy, mock, and script
handlers, not by statics.

Lifetimes (`internal/di`):

- `Container`: process-lifetime services (fs, output, server, request tracker).
- `Runtime`: everything built from one `UncorsConfig` (cache, HAR writers,
  routers, server targets). Anything config-derived that holds resources must
  be registered on the `Runtime` so a reload releases it.
- `Proxy`: serves one `Runtime` and swaps it on reload. Start, Restart,
  Shutdown, and Close are serialised.

Interactive mode (BubbleTea, `internal/uncors_app`) is the default;
`--interactive=false` runs headless (`internal/cli`). Both watch the config file
and call `Proxy.Restart`; an invalid file keeps the previous configuration.

## Conventions

- Handlers implement `contracts.Handler` (`ServeHTTP(w, r) error`), middleware
  implements `contracts.Middleware` (`ServeHTTP(w, r, next) error`). Wire new
  ones in `internal/handler/router` (default chain: `prepareDefaultHandler`)
  through a DI constructor.
- Constructors use the functional options pattern (`helpers.ApplyOptions`).
- Use the injected `afero.Fs`, not `os`, so tests can run on an in-memory fs.
- YAML tags are kebab-case (tagliatelle).
- Mocks are generated with minimock into `testing/mocks`; snapshots use
  go-snaps. Shared test helpers live in `testing/testutils`.
- Commit messages: `feat:`, `fix:`, `docs:`, `refactor:`, `test:`.

## Configuration

- `internal/config` loads CLI flags (pflag) and YAML (yaml.v3); flags override
  the file. Validation is plain Go code.
- `schema.json` is only for editors and `tests/schema`; it is not used at
  runtime. Keep it in sync with the config structs by hand.
- A new config option needs: the struct field, validation, a `schema.json`
  update, a fixture under `tests/schema/`, and docs in `docs/`.
- `--interactive` is a CLI flag only, not a YAML key.

## Documentation

`docs/*.md` is published to the GitHub wiki by the `publish-wiki` job in
`.github/workflows/release.yml`, so it must describe the current main branch.
When behavior changes, update the matching page; breaking changes since the
last release go in `docs/Migration-Guide.md`.

## Debugging

Set `UNCORS_LOGGING=/path/to/file` to append the internal log to a file; it is
discarded otherwise.
