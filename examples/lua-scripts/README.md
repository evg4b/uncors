# Lua Script Handler Example

This example demonstrates how to use the Lua script handler in UNCORS.

## Setup

1. Start the UNCORS server:
```bash
cd examples/lua-scripts
uncors
```

2. Test the endpoints using curl or your browser.

## Available Endpoints

### 1. Simple Hello World
```bash
curl http://localhost:3000/api/hello
```

Returns:
```json
{"message": "Hello from Lua!", "timestamp": "2025-10-20 01:30:45"}
```

### 2. Echo Service
```bash
curl -X POST http://localhost:3000/api/echo \
  -H "Content-Type: application/json" \
  -d '{"name": "Alice", "age": 30}'
```

Returns the request details including method, path, headers, and body.

### 3. Calculator
```bash
# Random number
curl "http://localhost:3000/api/calculate?operation=random&min=1&max=100"

# Square root
curl "http://localhost:3000/api/calculate?operation=sqrt&value=16"

# Power
curl "http://localhost:3000/api/calculate?operation=power&base=2&exponent=8"
```

### 4. User API (with path parameters)
```bash
curl http://localhost:3000/api/users/123
```

Returns:
```json
{
  "id": "123",
  "name": "User 123",
  "email": "user123@example.com",
  "created": "2025-10-20 01:30:45"
}
```

### 5. Authentication Check
```bash
# Without auth
curl http://localhost:3000/api/protected

# With auth
curl http://localhost:3000/api/protected \
  -H "Authorization: Bearer secret-token"
```

### 6. File-based Script
```bash
curl http://localhost:3000/api/complex
```

This endpoint uses a Lua script loaded from `scripts/complex.lua`.

## Files Structure

```
examples/lua-scripts/
├── .uncors.yaml          # Configuration file
├── scripts/
│   └── complex.lua       # External Lua script
└── README.md             # This file
```

## Testing Tips

- Use `curl -v` to see response headers
- Add `Origin` header to test CORS: `curl -H "Origin: http://example.com" ...`
- Try invalid operations to see error handling
