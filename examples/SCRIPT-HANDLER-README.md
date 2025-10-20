# Script Handler Examples

This directory contains examples demonstrating the Script Handler's **zero-buffering architecture** where Lua scripts write directly to Go's `http.ResponseWriter`.

## Key Concepts

### 🚀 Direct Write Architecture

All Lua operations write **immediately** to the HTTP connection:

```
Lua Script                  → ResponseWriter → HTTP Connection
    ↓                              ↓                 ↓
response:WriteString("x")    writer.Write()   Network Socket
```

**NO BUFFERING** - Data flows directly from Lua to the network!

### ⚠️ HTTP Protocol Rules

Since we write directly to `ResponseWriter`, you must follow HTTP protocol rules:

1. **Headers first** - Set all headers before writing body
2. **Status once** - `WriteHeader()` can only be called once
3. **No going back** - Cannot modify headers after body write starts

## Running Examples

```bash
# Start uncors with examples
uncors --config examples/script-handler-examples.yaml

# Test different endpoints
curl http://localhost:3000/api/simple
curl http://localhost:3000/api/stream
curl http://localhost:3000/api/user/123
curl -X POST http://localhost:3000/api/validate -d "test data"
```

## Example Breakdown

### ✅ Example 1: Correct Order (Simple)

```lua
-- 1. Headers first
response.headers["Content-Type"] = "application/json"

-- 2. Status code
response:WriteHeader(200)

-- 3. Body
response:WriteString('{"message": "Hello"}')
```

**Flow**: Headers set → Status sent → Body sent → Complete

### ✅ Example 2: Streaming Response

```lua
response:Header():Set("Content-Type", "text/plain")
response:WriteHeader(200)

-- Multiple writes - each goes to network immediately!
response:WriteString("Chunk 1\n")
response:WriteString("Chunk 2\n")
response:WriteString("Chunk 3\n")
```

**Flow**: Each `WriteString()` sends data to client in real-time!

### ❌ Example 9: Wrong Order (Don't Do This!)

```lua
-- BAD: Body first
response:WriteString("data")  -- This auto-sends WriteHeader(200)

-- BAD: Too late!
response.headers["Content-Type"] = "application/json"  -- IGNORED!
response:WriteHeader(404)  -- IGNORED!
```

**Problem**: Headers already sent with auto WriteHeader(200)

## API Reference

### Headers (Table Access)

```lua
-- Headers can be accessed as a table
response.headers["X-Key"] = "value"      -- Sets header
response:Header():Set("X-Key", "value")  -- Alternative method style
```

### Status & Body (Methods Only)

```lua
-- Status - method call only
response:WriteHeader(200)  -- Sends status (once!)

-- Body - method calls only
response:WriteString("text")  -- Sends body
response:Write("data")        -- Alternative
```

**Key difference**: Headers support both styles, but status/body require methods!

## Common Patterns

### Pattern 1: Simple JSON Response

```lua
response:Header():Set("Content-Type", "application/json")
response:WriteHeader(200)
response:WriteString('{"result": "success"}')
```

### Pattern 2: Error Handling

```lua
if not valid then
  response:Header():Set("Content-Type", "application/json")
  response:WriteHeader(400)
  response:WriteString('{"error": "invalid request"}')
  return
end

-- Success case
response:Header():Set("Content-Type", "application/json")
response:WriteHeader(200)
response:WriteString('{"result": "ok"}')
```

### Pattern 3: Streaming Large Data

```lua
response:Header():Set("Content-Type", "text/plain")
response:WriteHeader(200)

for i = 1, 1000 do
  response:WriteString("Line " .. i .. "\n")
  -- Data sent immediately, no buffering!
end
```

### Pattern 4: Dynamic Content Assembly

```lua
response:Header():Set("Content-Type", "application/json")
response:WriteHeader(200)

response:Write('{"items": [')
for i = 1, 10 do
  if i > 1 then response:Write(',') end
  response:Write('{"id": ')
  response:Write(tostring(i))
  response:Write('}')
end
response:Write(']}')
```

## Important Limitations

### No Direct Assignment to Status/Body

```lua
-- ❌ CANNOT READ
local status = response.status  -- Returns nil
local body = response.body      -- Returns nil

-- ❌ CANNOT WRITE DIRECTLY
response.status = 200   -- Silently ignored!
response.body = "text"  -- Silently ignored!

-- ✅ USE METHODS INSTEAD
response:WriteHeader(200)     -- Works!
response:WriteString("text")  -- Works!
```

### One-Time Operations

```lua
-- ❌ CANNOT CALL TWICE
response:WriteHeader(200)
response:WriteHeader(404)  -- Ignored! Already sent 200

-- ❌ CANNOT MODIFY HEADERS AFTER WRITE
response:WriteString("data")
response:Header():Set("X-Key", "val")  -- Ignored! Headers sent
```

## Performance Benefits

1. **Zero Memory Overhead**: No buffering = no memory allocation for response data
2. **Instant Transmission**: Data reaches client as soon as you write it
3. **True Streaming**: Perfect for large responses or server-sent events
4. **Minimal Latency**: No waiting for script to complete before sending

## Testing Your Scripts

Test scripts with curl to see direct writes in action:

```bash
# Watch streaming in real-time
curl -N http://localhost:3000/api/stream

# Check headers
curl -I http://localhost:3000/api/headers

# Test error handling
curl -v -X POST http://localhost:3000/api/validate
```

## Further Reading

- See `docs/9. Script Handler.md` for complete API documentation
- HTTP protocol: https://developer.mozilla.org/en-US/docs/Web/HTTP/Overview
- Go http.ResponseWriter: https://pkg.go.dev/net/http#ResponseWriter
