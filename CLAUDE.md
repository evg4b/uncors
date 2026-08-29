# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**UNCORS** is a lightweight local HTTP/HTTPS proxy that bypasses CORS restrictions by modifying CORS headers in responses. It's designed for development and testing workflows, supporting features like request mocking, response caching, request rewriting, static file serving, and HTTP Archive (HAR) traffic recording.

- Language: Go 1.24.1+
- Primary Use: Development proxy
- Key Package: `github.com/evg4b/uncors`

## Quick Start

### Build & Run
```bash
make build              # Build binary
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
./uncors generate-certs         # Generate TLS certificates
make format-docs                # Format markdown docs with Prettier
go mod tidy                     # Tidy dependencies
make upgrade                    # Upgrade all dependencies and run make all
```

## Architecture Overview

UNCORS follows a clean layered architecture with middleware composition:

### Request Flow
1. **Server** (`internal/server`) - TCP listener and request routing
2. **HAR Collector** (first middleware) - Non-blocking traffic recording
3. **Options Middleware** - CORS preflight request handling
4. **Cache Middleware** - In-memory response caching with TTL
5. **Handler Selection** - Route to Proxy, Mock, Script, or Static handler
6. **CORS Headers** - Add/modify response headers

### Core Packages

**`internal/uncors`** - Application lifecycle
- `Uncors` type: manages server startup, graceful shutdown, and config watching
- Watches for config file changes and restarts when needed

**`internal/config`** - Configuration loading & validation
- `LoadConfiguration()`: Parses CLI flags and YAML config file
- JSON Schema validation (schema.json)
- `ConfigWatcher`: File system watcher for live config reloads

**`internal/handler`** - Request routing and middleware
- **Proxy** - Forwards requests to upstream servers with modified CORS headers
- **Mock** - Returns predefined responses from files or config
- **Script** - Runs Lua scripts for dynamic responses (via gopher-lua)
- **Static** - Serves static files from filesystem
- **Middleware**: cache, rewrite, options, HAR collector

**`internal/contracts`** - Small, focused interfaces
- `Handler`: Interface for request handlers
- `Logger`: Logging abstraction
- `HTTPClient`: HTTP client contract

**`internal/infra`** - Infrastructure services
- HTTP client with connection pooling and proxy support
- Logger setup (`log/slog` to stderr or a file, level-controlled)
- TLS certificate generation and handling

**`internal/tui`** - Terminal UI and logging
- `CliOutput`: Colored console output with request/response formatting
- Request tracking and printing

**`internal/server`** - Server lifecycle
- `RequestTracker`: Tracks active requests for stats/logging
- `RequestPrinter`: Goroutine that prints request info

**`main.go`** - Entry point
- Interactive TUI mode (`-i` flag) via BubbleTea
- Non-interactive headless mode (default)
- Config watching and auto-restart
- Version checking and panic recovery

### HAR Collector Design
Located in `internal/handler/har`, implements non-blocking traffic recording:
- **Channel-based**: Entries sent over buffered channel (capacity 4096)
- **Async writes**: Single background goroutine handles disk I/O
- **Atomic file updates**: Write-to-temp-then-rename for data integrity
- **Per-mapping isolation**: Each mapping has its own `Writer` instance
- **Lifecycle**: Implements `io.Closer` for graceful shutdown
- **Security**: Excludes sensitive headers by default (Cookie, Authorization, etc.)

## Key Design Patterns

**Middleware Pattern**
```go
type Middleware = func(http.Handler) http.Handler
```
Composable request/response processing layers.

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
- Supports hot-reload without server restart

**Key Config Options**
- `proxy`: Upstream proxy URL (optional)
- `interactive`: Enable TUI mode
- `debug`: Enable debug logging
- `port`: Listen port (default: 3000)
- `mappings`: Array of request mappings (from/to hosts)

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
4. Add factory method in request handler routing
5. Add tests in `myhandler_test.go`

**New Middleware:**
1. Create package in `internal/handler/mymiddleware/`
2. Implement `func(http.Handler) http.Handler` signature
3. Update middleware chain in request handler
4. Add tests

**New Config Option:**
1. Add field to `internal/config/` struct
2. Update `schema.json` with validation rules
3. Add parser/validator if complex
4. Update CONTRIBUTING.md if user-facing

### Debugging
- Diagnostics go to stderr at `info` level; `--log-level debug`, `--log-file PATH` and `--quiet` control them (`--debug` is shorthand for `--log-level debug`)
- Run single test: `go test -run TestName ./internal/handler/proxy/`
- Race detector: Already enabled in `make test` and `make test-cover`
- Integration tests: `make test-integration` (slower, real network)

## Important Notes

**Scope**: UNCORS is a development-only tool. Security review is limited; not intended for production or remote proxy use.

**Performance**: Uses goroutines, connection pooling, in-memory caching, and the ristretto cache library for speed.

**Go Version**: Requires Go 1.24.1+ due to language features and dependency requirements.

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
│   ├── uncors/                  # App lifecycle & server management
│   ├── config/                  # Config loading, validation, watching
│   ├── handler/                 # Request handlers & middleware
│   │   ├── proxy/               # HTTP proxy handler
│   │   ├── mock/                # Mock response handler
│   │   ├── script/              # Lua script handler
│   │   ├── static/              # Static file handler
│   │   ├── cache/               # Response caching middleware
│   │   ├── har/                 # HAR traffic recording
│   │   ├── rewrite/             # URL/header rewriting
│   │   └── options/             # CORS preflight handling
│   ├── contracts/               # Interfaces (Handler, Logger, HTTPClient)
│   ├── infra/                   # HTTP client, logger, TLS
│   ├── server/                  # Server lifecycle & request tracking
│   ├── tui/                     # Terminal UI & colored output
│   ├── uncors_app/              # Interactive TUI app (BubbleTea)
│   ├── commands/                # CLI commands (generate-certs)
│   ├── version/                 # Version checking
│   ├── helpers/                 # Utilities
│   └── urlreplacer/             # URL replacement utility
├── testing/                     # Test mocks & helpers
├── tests/                       # Integration tests
└── docs/                        # User documentation (features, guides)
```

## Useful External Resources

- **Testing**: `stretchr/testify` for assertions, `minimock/v3` for mocks
- **CLI**: `spf13/pflag` for flags
- **YAML**: `goccy/go-yaml` and `gopkg.in/yaml.v3`
- **Lua**: `yuin/gopher-lua` and `layeh/gopher-json` for script handler
- **TUI**: `charm.land/bubbletea/v2`, `charm.land/lipgloss/v2`, `charm.land/bubbles/v2`
- **Caching**: `dgraph-io/ristretto/v2` for high-performance caching
