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
