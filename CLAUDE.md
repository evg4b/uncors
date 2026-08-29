# CLAUDE.md

Guidance for Claude Code (claude.ai/code) working in this repository.

**UNCORS** is a local development HTTP/HTTPS proxy that bypasses CORS
restrictions. It also mocks, caches, rewrites, serves static files, runs Lua
scripts, and records traffic to HAR archives.

- Module: `github.com/evg4b/uncors`
- Go version: see `go.mod` (currently 1.26)
- Scope: development tooling. Not intended for production or remote use.

## Structure

See [ARCHITECTURE.md](ARCHITECTURE.md) for the layering, the request flow and how
to extend things — it is the single description of the design, so add structural
facts there rather than duplicating them here.

For the current package list, run `go list ./...`; for what a package is for, run
`go doc ./internal/<pkg>`.

## Commands

```bash
make build              # compile every package (produces no binary)
make install            # install the binary into GOPATH/bin
make build-release      # build ./uncors with release flags

make test               # unit tests with the race detector
make test-integration   # real sockets and TLS (tagged `integration`)
make test-cover         # coverage.out
make dead-code          # unreachable functions under internal/

make format             # gofmt, gofumpt, golangci-lint --fix
make check              # format + test + dead-code + build
make format-docs        # prettier over the markdown

go test -run TestName ./internal/handler/proxy/
```

## Conventions

- **Standard net/http shapes.** Handlers are `http.Handler`; middleware is
  `func(http.Handler) http.Handler`. A handler that wants an error return uses
  `infra.HandlerFunc`, which renders the error with an appropriate status.
- **Functional options** for every constructor: `NewX(WithA(…), WithB(…))`.
- **Explicit dependencies.** A leaf package states what it needs (see
  `router.Deps`); it does not reach back into the container.
- **Diagnostics go through `log/slog`** to stderr. `contracts.Output` is for
  deliberate user-facing output (the logo, boxes, the request log) — a component
  should never have to choose between them.
- **Nothing on the request path may block on presentation.** Activity events are
  dropped, never queued, when a consumer is slow.
- **Config-derived resources belong to `di.Runtime`**, so a reload releases them.
- YAML keys are kebab-case (enforced by the `tagliatelle` linter).
- `.golangci.yml` enables all linters except the ones it lists; `make format`
  applies the fixable ones.

## Gotchas

- `make build` compiles but writes no binary; use `make build-release` or
  `make install` when you need one.
- Integration tests need the `integration` build tag, and they run the proxy
  against an in-memory filesystem — anything the proxy writes (HAR archives) is
  in `env.Fs`, not on disk.
- Snapshot tests use `go-snaps`; regenerate with `UPDATE_SNAPS=true go test …`
  and read the diff before committing it.
- `interactive` is a CLI flag only (`yaml:"-"`), and it falls back to plain
  output when stdout is not a terminal.
- `schema.json` is for editor completion; it is not used at runtime.
- Three documentation guards will fail a change that drifts: `tests/docs` loads
  every config example in `docs/`, `internal/cli` pins the documented flag table
  to the real flag set, and `make dead-code` rejects unreachable code.

## Before committing

1. `make format`
2. `make check`
3. Commit with a conventional prefix: `feat:`, `fix:`, `docs:`, `refactor:`,
   `test:`, `chore:`.
