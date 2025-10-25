# UNCORS Architecture

A quick overview of how UNCORS works and how the code is organized.

## What is UNCORS?

UNCORS is a local development proxy that bypasses CORS restrictions. It sits between your browser and backend servers, modifying CORS headers on the fly.

**How it works:**

- Intercepts HTTP/HTTPS requests
- Forwards requests to target servers
- Modifies CORS headers in responses
- Uses middleware for additional features (caching, mocking, etc.)

## Core Components

### Main Application (`internal/uncors`)

Manages server lifecycle, graceful shutdown, and config watching.

### Configuration (`internal/config`)

Loads and validates YAML config files using JSON Schema.

### Request Handlers (`internal/handler`)

Routes requests and builds middleware chains based on configuration.

**Available handlers:**

- **Proxy** - Forwards requests to upstream servers with modified CORS headers
- **Mock** - Returns predefined responses from files or config
- **Script** - Runs Lua scripts for dynamic responses
- **Static** - Serves static files from filesystem

**Middleware:**

- **Cache** - In-memory response caching with TTL
- **Rewrite** - URL/header/query parameter manipulation
- **Options** - Handles CORS preflight requests

### Infrastructure (`internal/infra`)

HTTP client, logger setup, TLS certificate handling.

### Terminal UI (`internal/tui`)

Colored logging and request/response formatting.

## Request Flow

1. **Client sends request** → UNCORS server
2. **Route matching** - Find mapping by host/port
3. **Middleware pipeline** - Apply options, rewrite, cache, static
4. **Handler selection** - Choose mock, script, or proxy handler
5. **CORS modification** - Add/modify CORS headers
6. **Response** - Return to client

Example CORS headers added:

```
Access-Control-Allow-Origin: *
Access-Control-Allow-Credentials: true
Access-Control-Allow-Methods: *
Access-Control-Allow-Headers: *
```

## Project Structure

```
uncors/
├── main.go
├── internal/
│   ├── config/           # Config loading & validation
│   ├── contracts/        # Interfaces (handler, logger, http client)
│   ├── handler/          # Request handlers & middleware
│   │   ├── cache/
│   │   ├── mock/
│   │   ├── proxy/
│   │   ├── script/
│   │   ├── static/
│   │   └── ...
│   ├── infra/            # HTTP client, logger, TLS
│   ├── tui/              # Terminal UI
│   ├── uncors/           # Main app
│   └── helpers/          # Utilities
├── testing/              # Mocks & test helpers
└── tests/                # Integration tests
```

## Key Design Patterns

**Middleware Pattern** - Composable request/response processing

```go
type Middleware = func(http.Handler) http.Handler
```

**Factory Pattern** - Creates handlers/middleware with dependency injection

```go
type ProxyHandlerFactory = func() contracts.Handler
```

**Interface-based** - Small interfaces for easy testing and mocking

## Extending UNCORS

Want to add a new feature? Here's where to start:

**New handler:**

1. Create package in `internal/handler/`
2. Implement `contracts.Handler` interface
3. Add factory to `RequestHandler`

**New middleware:**

1. Create middleware package
2. Implement `func(http.Handler) http.Handler` signature
3. Add to middleware chain

**New config option:**

1. Update structs in `internal/config/`
2. Update `schema.json`
3. Add validator if needed

## Testing

- Unit tests use mocks (generated with `minimock`)
- Integration tests in `tests/`
- Run: `make test` or `go test ./...`

## Important Notes

**Security:** UNCORS is for local development only. Don't expose it to the internet!

**Performance:** Uses goroutines, connection pooling, and in-memory caching for speed.
