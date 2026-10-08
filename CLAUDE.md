# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**UNCORS** is a lightweight local HTTP/HTTPS proxy that bypasses CORS restrictions by modifying CORS headers in responses. It's designed for development and testing workflows, supporting features like request mocking, response caching, request rewriting, static file serving, and HTTP Archive (HAR) traffic recording.

- Language: Go (version pinned in `go.mod`, currently 1.26.4)
- Primary Use: Development proxy
- Key Package: `github.com/evg4b/uncors`

## Quick Start

### Build & Run
```bash
make build-release      # Build the ./uncors binary (make build only compiles packages)
make install            # Install to GOPATH/bin
./uncors --from 'http://localhost:8080' --to 'https://github.com'
```

### Testing
```bash
make test               # Run all unit tests with race detection
make test-integration   # Run integration tests (real sockets + TLS)
make test-cover         # Generate coverage report (coverage.out)
go test -run TestName ./... # Run specific test by name pattern
```

### Code Quality
```bash
make check              # Run format + test + build (full validation)
make format             # Run gofmt, gofumpt, and golangci-lint --fix
golangci-lint run       # Manual linting (uses .golangci.yml)
```

### Configuration & Generation
```bash
./uncors generate-certs         # Generate the local CA in ~/.config/uncors
make format-docs                # Format markdown docs with Prettier
go mod tidy                     # Tidy dependencies
make upgrade                    # Upgrade all dependencies and run make all
```

## Architecture Overview

UNCORS follows a clean layered architecture with middleware composition:

### Request Flow
1. **Server** (`internal/server`) - one listener per port/scheme group of mappings
2. **Router** (`internal/handler/router`, gorilla/mux) - picks the mapping by host name, then the first matching route:
   statics (path prefix) → mocks → scripts → rewrites → mapping default handler
3. **Default handler chain** - HAR collector → cache → OPTIONS handling → proxy
4. **CORS Headers** - added by the proxy, mock, and script handlers (not by statics)

Statics and rewrites pass unhandled requests to the default handler, so only
proxied traffic (including cache hits and OPTIONS) is cached and recorded to HAR;
mock and script responses are not.

### Core Packages

**`internal/cli`** - Command entry points
- `RunUncors()`: loads the configuration and runs interactive or headless mode
- `GenerateCerts()`: the `generate-certs` sub-command

**`internal/di`** - Dependency container & configuration generations
- `Container`: process-lifetime services (fs, output, server, request tracker)
- `Runtime`: everything derived from one `UncorsConfig` (cache, HAR writers,
  routers, server targets); closing it releases exactly that generation
- `Proxy`: serves one `Runtime` on the server and swaps generations on reload

**`internal/config`** - Configuration loading & validation
- `LoadConfiguration()`: Parses CLI flags (pflag) and the YAML file (yaml.v3), validates in Go code
- `schema.json` is for editors and `tests/schema` only; it is not used at runtime, keep it in sync by hand
- `Watcher`: fsnotify watcher; the CLI reloads config and calls `di.Proxy.Restart` on change

**`internal/handler`** - Request routing and middleware
- **Proxy** - Forwards requests to upstream servers with modified CORS headers
- **Mock** - Returns predefined responses from files or config
- **Script** - Runs Lua scripts for dynamic responses (via gopher-lua)
- **Static** - Serves static files from filesystem
- **Middleware**: cache, rewrite, options, HAR collector

**`internal/contracts`** - Small, focused interfaces
- `Handler`: `ServeHTTP(writer, request) error`
- `Middleware`: `ServeHTTP(writer, request, next) error`
- `Logger`: Logging abstraction
- `HTTPClient`: HTTP client contract

**`internal/infra`** - Infrastructure services
- HTTP client with connection pooling and proxy support
- Logger setup (appends to the file in UNCORS_LOGGING; discarded when unset)
- CORS header helpers and HTTP error pages
- (TLS and the local CA live in `internal/server`)

**`internal/tui`** - Terminal UI and logging
- `CliOutput`: Colored console output with request/response formatting
- Request tracking and printing

**`internal/server`** - Server lifecycle
- `RequestTracker`: Tracks active requests for stats/logging
- `RequestPrinter`: Goroutine that prints request info

**`main.go`** - Entry point
- Builds the DI container, dispatches to `generate-certs` or `cli.RunUncors`
- Panic recovery and error reporting
- Interactive TUI (BubbleTea) is the default; `--interactive=false` gives headless mode
  (mode selection, config watching, and version checking live in `internal/cli`)

### HAR Collector Design
Located in `internal/handler/har`, implements non-blocking traffic recording:
- **Channel-based**: Entries sent over buffered channel (capacity 4096)
- **Async writes**: Single background goroutine handles disk I/O
- **Atomic file updates**: Write-to-temp-then-rename for data integrity
- **Per-mapping isolation**: Each mapping has its own `Writer` instance
- **Lifecycle**: Implements `io.Closer`; closed with its `di.Runtime` on shutdown/reload, and a new runtime starts an empty recording that overwrites the file
- **Scope**: Sits in the default handler chain, so mock/script/served-static responses are not recorded
- **Security**: Excludes sensitive headers by default (Cookie, Authorization, etc.)

## Key Design Patterns

**Middleware Pattern**
```go
type Middleware interface {
	ServeHTTP(writer ResponseWriter, request *Request, next Next) error
}
```
Composable request/response processing layers (`internal/contracts`).

**Factory Pattern**
Handlers/middleware created with dependency injection (options pattern in Go).

**Interface-based Design**
Small, focused interfaces (`Handler`, `Logger`, `HTTPClient`) enable easy testing and mocking.

## Testing Strategy

- **Unit Tests**: Co-located with code in `*_test.go` files, using minimock for mocks
- **Integration Tests**: In `tests/integration/` tagged with `// +build integration`
- **Mocks**: Generated with `gojuno/minimock/v3`
- **Snapshots**: Using `gkampitakis/go-snaps` for golden file testing

Key test flags:
- `-race`: Detects data races (always enabled in make test)
- `-tags integration`: Runs integration tests that use real sockets/TLS
- `-tags release`: Runs release-specific tests
- `-timeout 1m`: Sets timeout for long-running tests

## Configuration Structure

**Config Loading** (`internal/config`)
- CLI flags override YAML file settings
- YAML is validated against `schema.json` (JSON Schema)
- Config watcher uses `fsnotify` for file system events
- Hot reload: a changed file builds a new `di.Runtime` and restarts the server with it; an invalid file keeps the old config

**Key Config Options**
- `proxy`: Upstream proxy URL (optional)
- `cache-config`: `expiration-time`, `max-size`, `methods`
- `mappings`: Array of request mappings (from/to hosts); the listen port comes from each `from` URL
- `--interactive` is a CLI flag only (default true), not a YAML key

## Development Workflow

### Before Committing
1. Run `make format` to auto-fix code style
2. Run `make test` to verify tests pass
3. Run `make check` for comprehensive validation
4. Commit with clear message: `feat:`, `fix:`, `docs:`, `refactor:`, `test:`

### Adding a New Feature

**New Handler Type:**
1. Create `internal/handler/myhandler/` package
2. Implement `contracts.Handler` interface
3. Update config schema in `schema.json`
4. Add a DI constructor and register routes in `internal/handler/router`
5. Add tests in `myhandler_test.go`

**New Middleware:**
1. Create package in `internal/handler/mymiddleware/`
2. Implement `contracts.Middleware`
3. Wire it in `internal/handler/router` (usually `prepareDefaultHandler`)
4. Add tests

**New Config Option:**
1. Add field to `internal/config/` struct
2. Update `schema.json` with validation rules
3. Add parser/validator if complex
4. Add a fixture under `tests/schema/` and document it in `docs/`

### Debugging
- Enable logging: Set `UNCORS_LOGGING=/path/to/logfile` environment variable
- Run single test: `go test -run TestName ./internal/handler/proxy/`
- Race detector: Already enabled in `make test` and `make test-cover`
- Integration tests: `make test-integration` (slower, real network)

## Important Notes

**Scope**: UNCORS is a development-only tool. Security review is limited; not intended for production or remote proxy use.

**Performance**: Uses goroutines, connection pooling, in-memory caching, and the ristretto cache library for speed.

**Go Version**: See `go.mod`.

**Linux Compatibility**: Primarily developed on macOS; runs on Linux and Windows.

**Code Style**: Follows Go conventions. Key linters enabled in `.golangci.yml`:
- Default: all linters except those listed in `disable:`
- YAML tags use kebab-case (tagliatelle rule)
- Variable naming lenience for common short names (fs, to, ok, form, ca)

## File Structure

```
uncors/
├── main.go                      # Entry point (CLI, TUI mode selection, config loading)
├── main_test.go                 # Main function tests
├── schema.json                  # JSON Schema for config validation
├── ARCHITECTURE.md              # Detailed architecture docs
├── CONTRIBUTING.md              # Contribution guidelines
├── Makefile                     # Build automation
├── .golangci.yml                # Linter configuration
├── go.mod / go.sum              # Dependencies
├── internal/
│   ├── cli/                     # Command entry points (run, generate-certs)
│   ├── di/                      # Container, config generations, proxy lifecycle
│   ├── config/                  # Config loading, validation, watching
│   ├── handler/                 # Request handlers & middleware
│   │   ├── proxy/               # HTTP proxy handler
│   │   ├── mock/                # Mock response handler
│   │   ├── script/              # Lua script handler
│   │   ├── static/              # Static file handler
│   │   ├── cache/               # Response caching middleware
│   │   ├── har/                 # HAR traffic recording
│   │   ├── rewrite/             # Path and upstream-host rewriting
│   │   ├── options/             # CORS preflight handling
│   │   └── router/              # gorilla/mux routing per mapping
│   ├── contracts/               # Interfaces (Handler, Middleware, Logger, HTTPClient)
│   ├── infra/                   # HTTP client, CORS helpers, error pages, logging
│   ├── server/                  # Listeners, TLS/CA, request tracking
│   ├── tui/                     # Terminal UI & colored output
│   ├── uncors_app/              # Interactive TUI app (BubbleTea)
│   ├── commands/                # CLI commands (generate-certs)
│   ├── version/                 # Version checking
│   ├── helpers/                 # Utilities
│   └── urlreplacer/             # Host matching with {name} placeholders
├── pkg/urlt/                    # Fork of net/url with placeholder support
├── testing/                     # Test mocks & helpers
├── tests/                       # Integration tests and schema tests
└── docs/                        # User documentation (features, guides)
```

## Useful External Resources

- **Testing**: `stretchr/testify` for assertions, `minimock/v3` for mocks
- **CLI**: `spf13/pflag` for flags
- **YAML**: `gopkg.in/yaml.v3`
- **Lua**: `yuin/gopher-lua` and `layeh/gopher-json` for script handler
- **TUI**: `charm.land/bubbletea/v2`, `charm.land/lipgloss/v2`, `charm.land/bubbles/v2`
- **Caching**: `dgraph-io/ristretto/v2` for high-performance caching
