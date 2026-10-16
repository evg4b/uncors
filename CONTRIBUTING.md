# Contributing to UNCORS

Contributions are welcome. UNCORS is a pet project, so the process is kept
simple.

## Reporting bugs

Open an issue with:

- what you expected to happen;
- what happened instead;
- steps to reproduce;
- your UNCORS version (`uncors --version`), operating system, and
  configuration, if relevant.

## Suggesting features

Open an issue that describes the problem you want to solve, the solution you
have in mind, and why it would be useful.

## Pull requests

1. Fork the repository and create a branch from `main`.
2. Make your changes and add tests where it makes sense.
3. Run `make check`.
4. Open a pull request.

## Development setup

You need:

- Go, at the version in `go.mod` or newer;
- Make;
- [gum](https://github.com/charmbracelet/gum), which the Makefile uses for its
  output;
- [gofumpt](https://github.com/mvdan/gofumpt) and
  [golangci-lint](https://golangci-lint.run/) for `make format`.

```bash
git clone https://github.com/YOUR_USERNAME/uncors.git
cd uncors
go mod download
make test
```

To run a build of your working tree with tracing, put a config in
`.uncors.yaml` (it is git-ignored) and start it with
[otelop](https://github.com/mashiro/otelop) as the trace viewer:

```bash
uncors generate-certs   # once, if the config has https mappings
docker compose up --build
```

Traces, logs, and metrics are at <http://localhost:4319>. `compose.yaml`
publishes ports 3000, 4000, and 2000; change them there if your mappings use
other ports.

## Make targets

```bash
make check            # format, test, build
make test             # unit tests with the race detector
make test-integration # end-to-end tests with real sockets and TLS
make test-cover       # tests with a coverage report in coverage.out
make format           # gofmt, gofumpt, golangci-lint --fix
make build            # compile all packages
make build-release    # build the uncors binary
make install          # install the binary into GOPATH/bin
make format-docs      # format Markdown with Prettier
make playground       # pick a TUI story and watch it while you edit
make stories          # update the TUI story snapshots
```

The same without Make:

```bash
go test -race ./...
go build -tags release .
golangci-lint run
```

## Code style

- Follow standard Go conventions.
- Run `make format` before committing.
- The linter configuration is in [.golangci.yml](.golangci.yml).
- If you change the configuration format, update `schema.json`, the fixtures
  in `tests/schema`, and the pages in `docs/`.

## Tests

Put unit tests in `_test.go` files next to the code. Integration tests live in
`tests/integration` and use the `integration` build tag.

## TUI stories

`internal/uncors_app/stories.go` holds stories: each widget, and the whole app,
in one state with fixed data (`tracker/many`, `app/notices`, …). They work
like Storybook stories:

- `make playground` asks which story to show (type to filter), then shows it
  across the whole terminal. Keys go to the story; `q` quits. Every saved Go
  file under `internal`, `pkg`, or `tools` rebuilds it and reopens the same
  story; resize the terminal to see other widths. Pass `STORY=tracker/many`
  to skip the question.
- `TestStories` snapshots every story at 40, 80, and 120 columns, cut the way
  Bubble Tea cuts long lines. Each snapshot has the plain text, for a readable
  diff, and the same frame with its colours. Run `make stories` after a
  deliberate change to the look, and review the diff.
- `go run ./tools/tuiplay -render tracker/many -width 60` prints one frame;
  `-plain` drops the colours.

When you add a widget or a new state, add a story for it. Stories must not
depend on the clock, the machine, or map order, or the snapshots will flake.

## Commit messages

Use a short prefix:

```
feat: add new feature
fix: fix the bug
docs: update documentation
refactor: improve code structure
test: add tests
```

Reference issues when relevant, for example `Fixes #123`.

## Questions

Open an issue or read the [wiki](https://github.com/evg4b/uncors/wiki).
