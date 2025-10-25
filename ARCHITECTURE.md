# UNCORS Architecture

This document provides an overview of the UNCORS architecture, its key components, and how they work together.

## Table of Contents

- [Overview](#overview)
- [High-Level Architecture](#high-level-architecture)
- [Core Components](#core-components)
- [Request Flow](#request-flow)
- [Package Structure](#package-structure)
- [Key Design Patterns](#key-design-patterns)
- [Extension Points](#extension-points)

## Overview

UNCORS is a development HTTP/HTTPS proxy designed to bypass CORS restrictions during local development. It acts as a reverse proxy that intercepts requests, modifies CORS headers, and forwards them to target servers.

### Key Characteristics

- **Reverse Proxy Architecture**: Intercepts and forwards HTTP/HTTPS requests
- **Middleware-Based**: Uses composable middleware for request/response processing
- **Configuration-Driven**: Flexible YAML-based configuration
- **Handler Pipeline**: Chain of handlers for different request types

## Core Components

### 1. Application (`internal/uncors`)

The main application orchestrator that:
- Manages server lifecycle (start, stop, restart)
- Handles graceful shutdown
- Watches configuration changes
- Coordinates HTTP/HTTPS servers

**Key Files:**
- `app.go`: Main application structure
- `server.go`: HTTP server management

### 2. Configuration (`internal/config`)

Handles configuration loading, parsing, and validation:
- Loads from YAML files or CLI flags
- Validates configuration using JSON Schema
- Provides configuration structures

**Key Components:**
- `loader.go`: Configuration loading logic
- `validators/`: Configuration validation rules
- `structs.go`: Configuration data structures

### 3. Request Handler (`internal/handler`)

Core request processing logic with factory pattern:
- Routes requests based on host mappings
- Builds middleware chains
- Selects appropriate handlers

**Structure:**
```go
type RequestHandler struct {
    *mux.Router
    logger   contracts.Logger
    mappings config.Mappings

    // Factory functions for handlers/middleware
    cacheMiddlewareFactory   CacheMiddlewareFactory
    staticMiddlewareFactory  StaticMiddlewareFactory
    proxyHandlerFactory      ProxyHandlerFactory
    mockHandlerFactory       MockHandlerFactory
    scriptHandlerFactory     ScriptHandlerFactory
    rewriteMiddlewareFactory RewriteMiddlewareFactory
    optionsMiddlewareFactory OptionsMiddlewareFactory
}
```

### 4. Handlers (Handler Implementations)

#### Proxy Handler (`internal/handler/proxy`)
- Forwards requests to upstream servers
- Modifies CORS headers in responses
- Handles secure cookies
- Supports HTTP/HTTPS proxying

#### Mock Handler (`internal/handler/mock`)
- Returns predefined responses
- Supports file-based responses
- Configurable status codes and headers
- Optional response delays

#### Script Handler (`internal/handler/script`)
- Executes Lua scripts for dynamic responses
- Provides JSON manipulation capabilities
- Access to request context
- Flexible response generation

#### Static Handler (`internal/handler/static`)
- Serves static files from filesystem
- Supports directory listings
- Content-type detection
- Path mapping

### 5. Middleware

#### Cache Middleware (`internal/handler/cache`)
- In-memory response caching
- Glob-based cache key matching
- TTL support
- Cache invalidation

#### Rewrite Middleware (`internal/handler/rewrite`)
- URL path rewriting
- Query parameter manipulation
- Header modification

#### Options Middleware (`internal/handler/options`)
- Handles OPTIONS preflight requests
- Configurable CORS headers
- Custom OPTIONS responses

### 6. Infrastructure (`internal/infra`)

Provides infrastructure concerns:
- HTTP client creation with proxy support
- Logger configuration
- TLS/SSL certificate handling

### 7. URL Processing (`internal/urlparser`, `internal/urlreplacer`)

- **URL Parser**: Parses and validates URLs from configuration
- **URL Replacer**: Performs URL transformations and replacements

### 8. Terminal UI (`internal/tui`)

Provides rich terminal output:
- Colored logging
- Request/response formatting
- Logo and banners
- Theme support (light/dark)

## Request Flow

### 1. Request Reception

```
Client → UNCORS Server (Port 3000) → Gorilla Mux Router
```

### 2. Host Mapping Resolution

```go
// Example mapping configuration
mappings:
  - from: http://localhost:3000
    to: https://api.example.com
```

The handler matches incoming requests to configured mappings using:
- Exact host matching
- Wildcard patterns (`*.example.com`)
- Port matching

### 3. Middleware Pipeline Execution

Middleware is applied in order:
1. **Options Middleware**: Handles preflight requests
2. **Rewrite Middleware**: Rewrites URLs/headers
3. **Cache Middleware**: Checks cache for responses
4. **Static Middleware**: Serves static files if configured

### 4. Handler Selection

Based on configuration, one handler is selected:
- **Mock Handler**: If response mocking is configured
- **Script Handler**: If Lua script is configured
- **Proxy Handler**: Default, forwards to upstream

### 5. CORS Header Modification

For proxy responses, CORS headers are modified:
```go
// Added/Modified headers
Access-Control-Allow-Origin: *
Access-Control-Allow-Credentials: true
Access-Control-Allow-Methods: *
Access-Control-Allow-Headers: *
```

### 6. Response Return

Modified response is returned to the client.

## Package Structure

```
uncors/
├── main.go                 # Application entry point
├── internal/
│   ├── config/            # Configuration management
│   │   ├── loader.go      # Config loading
│   │   ├── structs.go     # Config structures
│   │   └── validators/    # Validation rules
│   │       ├── base/      # Base validators
│   │       └── *.go       # Specific validators
│   │
│   ├── contracts/         # Interfaces and contracts
│   │   ├── handler.go     # Handler interface
│   │   ├── http_client.go # HTTP client interface
│   │   └── logger.go      # Logger interface
│   │
│   ├── handler/           # Request handling
│   │   ├── uncors_handler.go     # Main handler
│   │   ├── cache/                # Cache middleware
│   │   ├── mock/                 # Mock responses
│   │   ├── options/              # OPTIONS handling
│   │   ├── proxy/                # Proxy functionality
│   │   ├── rewrite/              # URL rewriting
│   │   ├── script/               # Lua scripting
│   │   └── static/               # Static files
│   │
│   ├── helpers/           # Utility functions
│   │   ├── closer.go      # Resource cleanup
│   │   ├── functions.go   # General helpers
│   │   └── graceful.go    # Graceful shutdown
│   │
│   ├── infra/             # Infrastructure
│   │   ├── http_client.go # HTTP client factory
│   │   └── logger.go      # Logger setup
│   │
│   ├── tui/               # Terminal UI
│   │   ├── logo.go        # ASCII logo
│   │   ├── request.go     # Request formatting
│   │   └── styles/        # Color themes
│   │
│   ├── uncors/            # Main app logic
│   │   ├── app.go         # Application struct
│   │   └── server.go      # Server management
│   │
│   ├── urlparser/         # URL parsing
│   ├── urlreplacer/       # URL replacement
│   └── version/           # Version checking
│
├── testing/               # Test utilities
│   ├── mocks/            # Generated mocks
│   ├── testutils/        # Test helpers
│   └── hosts/            # Test host utilities
│
└── tests/                # Integration tests
    └── schema/           # Schema validation tests
```

## Key Design Patterns

### 1. Factory Pattern

Used extensively for creating handlers and middleware:

```go
type ProxyHandlerFactory = func() contracts.Handler
type CacheMiddlewareFactory = func(globs config.CacheGlobs) contracts.Middleware
```

Benefits:
- Testability through dependency injection
- Flexibility in handler creation
- Easy mocking for tests

### 2. Middleware Pattern

Composable request/response processing:

```go
type Middleware = func(http.Handler) http.Handler
```

Allows:
- Separation of concerns
- Request/response modification
- Pipeline composition

### 3. Options Pattern

Used for flexible object construction:

```go
type RequestHandlerOption func(*RequestHandler)

func WithLogger(logger contracts.Logger) RequestHandlerOption {
    return func(h *RequestHandler) {
        h.logger = logger
    }
}
```

Benefits:
- Optional configuration
- Backward compatibility
- Clean API

### 4. Interface Segregation

Small, focused interfaces for testability:

```go
type Logger interface {
    Info(msg string)
    Error(msg string)
    Debug(msg string)
}

type Handler interface {
    ServeHTTP(w http.ResponseWriter, r *http.Request)
}
```

### 5. Dependency Injection

All dependencies are injected via constructors or options:

```go
func NewUncorsRequestHandler(options ...RequestHandlerOption) *RequestHandler
```

Enables:
- Easy testing with mocks
- Flexible configuration
- Loose coupling

## Extension Points

### Adding New Handlers

1. Create handler package in `internal/handler/`
2. Implement `contracts.Handler` interface
3. Add factory function type to `RequestHandler`
4. Register in handler pipeline

Example:
```go
type CustomHandlerFactory = func(config CustomConfig) contracts.Handler

// In RequestHandler
customHandlerFactory CustomHandlerFactory
```

### Adding New Middleware

1. Create middleware package
2. Implement middleware function signature:
   ```go
   func MyMiddleware(config Config) contracts.Middleware
   ```
3. Add to middleware chain in `RequestHandler`

### Configuration Extension

1. Add fields to `config.Mapping` or create new config struct
2. Update JSON schema in `schema.json`
3. Add validators in `internal/config/validators/`

## Testing Strategy

### Unit Tests
- Mock external dependencies using interfaces
- Use `minimock` for mock generation
- Table-driven tests for multiple scenarios

### Integration Tests
- Full application lifecycle testing
- Real HTTP requests/responses
- Configuration validation tests

### Test Utilities
- `testing/testutils`: Common test helpers
- `testing/mocks`: Generated mocks
- `testing/hosts`: Host resolution helpers

## Performance Considerations

### Caching
- In-memory cache with TTL
- Glob-based cache key matching
- Automatic cache invalidation

### Concurrency
- Goroutine per request
- Mutex-protected shared state
- Context-based cancellation

### Resource Management
- Connection pooling for upstream requests
- Graceful shutdown for active connections
- Defer-based cleanup

## Security Considerations

### Development Only
UNCORS is designed for development environments and should not be exposed to the internet.

### TLS/SSL
- Supports HTTPS proxying
- Certificate generation for local development
- Proper certificate validation

### Input Validation
- Configuration validation using JSON Schema
- URL validation and sanitization
- Path traversal protection in static handler

## Future Architecture Improvements

Potential enhancements mentioned in the roadmap:
- Content URL replacement (HTML, JSON)
- HAR file export for request/response recording
- WebSocket proxying support
- Plugin system for extensibility

## References

- [Go Documentation](https://golang.org/doc/)
- [Gorilla Mux](https://github.com/gorilla/mux)
- [Viper Configuration](https://github.com/spf13/viper)
- [Reverse Proxy Design](https://en.wikipedia.org/wiki/Reverse_proxy)
