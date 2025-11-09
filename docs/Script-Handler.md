The script handler allows you to implement custom request handling logic using scripts. This provides maximum flexibility for generating dynamic responses, implementing custom business logic, or creating complex API simulations during development and testing.

**Key features:**

- **Inline or file-based scripts**: Define script code directly in configuration or load from external files
- **Request access**: Full access to request properties (method, URL, headers, body, query parameters)
- **Response control**: Set status codes, headers, and body content from script
- **Standard libraries**: Use math, string, table, OS, and JSON libraries
- **Path-based matching**: Define which URLs to handle with scripts
- **Method-specific**: Target specific HTTP methods (GET, POST, etc.)
- **Query parameter filtering**: Match requests with specific query strings
- **Header matching**: Filter by HTTP headers

**Configuration structure:**

```yaml
mappings:
  - from: ...
    to: ...
    scripts:
      - path: /api/custom
        method: POST
        queries:
          param1: value1
        headers:
          Content-Type: application/json
        script: |
          response.headers["Content-Type"] = "application/json"
          response:WriteHeader(200)
          response:WriteString('{"message": "Hello from script"}')
        file: /path/to/script.lua # Alternative to inline script
```

# Request Matching

Configure which requests should be handled by the script:

## Path (Required)

Defines the URL path to handle. Supports static paths and variable segments.

**Examples:**

```yaml
path: /api/custom             # Static path
path: /users/{id}             # Variable segment
path: /posts/{postId}/data    # Multiple variables
```

Variable segments (e.g., `{id}`) match any value in that position. A request to `/users/123` matches `/users/{id}`.

## Method (Optional)

Specifies the HTTP method to match.

| Property | Type   | Default | Description                                                |
| -------- | ------ | ------- | ---------------------------------------------------------- |
| `method` | string | Any     | HTTP method: `GET`, `POST`, `PUT`, `DELETE`, `PATCH`, etc. |

If omitted, the script matches all HTTP methods.

## Query Parameters (Optional)

Match requests with specific query string parameters.

```yaml
queries:
  param1: value1
  param2: value2
```

If omitted, all query parameter combinations are matched.

## Headers (Optional)

Match requests with specific HTTP headers.

```yaml
headers:
  Content-Type: application/json
  Authorization: Bearer token123
```

If omitted, all header combinations are matched.

# Script Configuration

Define the script to execute when a request is matched.

## Script Properties

| Property | Type   | Required      | Description                    |
| -------- | ------ | ------------- | ------------------------------ |
| `script` | string | Conditional\* | Inline script code             |
| `file`   | string | Conditional\* | Path to file containing script |

**\*Either `script` or `file` must be specified, but not both.**

## Inline Script

Define script code directly in the configuration:

```yaml
scripts:
  - path: /api/greeting
    method: GET
    script: |
      local name = request.query_params["name"] or "World"
      response.headers["Content-Type"] = "application/json"
      response:WriteHeader(200)
      response:WriteString('{"message": "Hello, ' .. name .. '"}')
```

## File-based Script

Load script code from an external file:

```yaml
scripts:
  - path: /api/calculate
    method: POST
    file: ~/scripts/calculator.lua
```

**File example (`~/scripts/calculator.lua`):**

```lua
-- Access request body
local body = request.body

-- Parse and process (example)
local result = 42  -- Your calculation logic here

-- Set response
response.headers["Content-Type"] = "application/json"
response:WriteHeader(200)
response:WriteString('{"result": ' .. result .. '}')
```

# Request Object

The `request` object provides access to incoming HTTP request properties.

## Request Properties

| Property       | Type   | Description                                     | Example                             |
| -------------- | ------ | ----------------------------------------------- | ----------------------------------- |
| `method`       | string | HTTP method                                     | `"GET"`, `"POST"`, etc.             |
| `url`          | string | Full request URL                                | `"http://localhost/api/users?id=1"` |
| `path`         | string | URL path                                        | `"/api/users"`                      |
| `query`        | string | Raw query string                                | `"id=1&name=test"`                  |
| `host`         | string | Host header value                               | `"localhost:8080"`                  |
| `remote_addr`  | string | Client IP address                               | `"127.0.0.1:12345"`                 |
| `body`         | string | Request body content                            | `'{"data": "value"}'`               |
| `headers`      | table  | Request headers (table with string keys/values) | `request.headers["Content-Type"]`   |
| `query_params` | table  | Parsed query parameters                         | `request.query_params["id"]`        |
| `path_params`  | table  | Path parameters from route                      | `request.path_params["id"]`         |

## Accessing Request Data

### HTTP Method

```lua
if request.method == "GET" then
    response:WriteString("This is a GET request")
end
```

### URL and Path

```lua
response:WriteString("You accessed: " .. request.path)
```

### Headers

```lua
local contentType = request.headers["Content-Type"]
local userAgent = request.headers["User-Agent"]

response:WriteString("Content-Type: " .. (contentType or "not set"))
```

### Query Parameters

```lua
local id = request.query_params["id"]
local filter = request.query_params["filter"]

if id then
    response:WriteString('{"id": ' .. id .. '}')
else
    response:WriteHeader(400)
    response:WriteString('{"error": "Missing id parameter"}')
end
```

### Path Parameters

Path parameters are extracted from the URL route pattern using wildcards (`{param}`). For example, if your route is `/users/{id}/posts/{postId}`, you can access these parameters:

```lua
local userId = request.path_params["id"]
local postId = request.path_params["postId"]

if userId and postId then
    response:WriteString('{"user": ' .. userId .. ', "post": ' .. postId .. '}')
else
    response:WriteHeader(400)
    response:WriteString('{"error": "Missing path parameters"}')
end
```

**Configuration example:**

```yaml
scripts:
  - path: /users/{id}/posts/{postId}
    method: GET
    script: |
      local userId = request.path_params["id"]
      local postId = request.path_params["postId"]
      response.headers["Content-Type"] = "application/json"
      response:WriteHeader(200)
      response:WriteString('{"user": "' .. userId .. '", "post": "' .. postId .. '"}')
```

### Request Body

```lua
local body = request.body

-- Simple body echo
response:WriteString("You sent: " .. body)

-- Or process the body
if string.find(body, "error") then
    response:WriteHeader(500)
else
    response:WriteHeader(200)
end
```

### Host Information

```lua
if request.host == "api.example.com" then
    response:WriteString("Production API")
else
    response:WriteString("Development API")
end
```

# Response Object

The `response` object controls the HTTP response returned to the client.

## Response Properties

| Property  | Type  | Access     | Description                                      |
| --------- | ----- | ---------- | ------------------------------------------------ |
| `headers` | table | Read/Write | Response headers (table with string keys/values) |

**Note:** `response.status` and `response.body` are not accessible as properties. Use methods instead:

- Use `response:WriteHeader(code)` to set status
- Use `response:Write(data)` or `response:WriteString(str)` to write body

## Response API

The script handler provides a method-based API that mirrors Go's `http.ResponseWriter` interface:

```lua
-- Set headers (table access)
response.headers["Content-Type"] = "application/json"

-- Set status (method call)
response:WriteHeader(200)

-- Write body (method call)
response:WriteString('{"message": "Hello"}')
```

**Key points:**

- **Headers**: Direct table access (`response.headers["Name"] = "value"`)
- **Status & Body**: Method-based only (`response:WriteHeader()`, `response:Write()`, `response:WriteString()`)
- **Cannot read**: `response.status` and `response.body` return `nil` if read

### Internal Architecture

**ZERO BUFFERING - Direct Write to HTTP Connection:**

All Lua operations write **directly** to Go's `http.ResponseWriter` without any intermediate buffering:

- **True streaming**: Data flows immediately to the HTTP connection during script execution
- **No buffering**: No intermediate storage - what you write in Lua goes straight to the network
- **Go semantics**: Same rules as Go's `http.ResponseWriter`:
  - Headers must be set before first write
  - `WriteHeader()` can only be called once
  - Headers cannot be modified after first write to body
- **User responsibility**: You must call methods in correct order (headers → WriteHeader → Write)

```
Lua Script                   Go Runtime              Network
   ↓                            ↓                       ↓
response:WriteString("x") → writer.Write(...) → HTTP Connection
response:Write("data")    → writer.Write(...) → HTTP Connection
response:WriteHeader(200) → writer.WriteHeader() → HTTP Headers Sent
```

### Important Notes

⚠️ **No direct assignment**: Cannot assign to `response.status` or `response.body` (silently ignored)
⚠️ **Cannot read**: `response.status` and `response.body` return `nil` if read
⚠️ **Order matters**: Call methods in correct HTTP order or behavior is undefined
⚠️ **Auto-header**: If you write body without calling `WriteHeader()`, status 200 is sent automatically

## Go-style Methods

| Method                       | Description                    | Example                                      |
| ---------------------------- | ------------------------------ | -------------------------------------------- |
| `response:WriteHeader(code)` | Set HTTP status code           | `response:WriteHeader(200)`                  |
| `response:Write(data)`       | Append data to response body   | `response:Write("Hello")`                    |
| `response:WriteString(str)`  | Append string to response body | `response:WriteString("World")`              |
| `response:Header()`          | Get headers object             | `response:Header():Set("X-Custom", "value")` |

### Header Methods

The `response:Header()` method returns a headers object with these methods:

| Method            | Description        | Example                                                     |
| ----------------- | ------------------ | ----------------------------------------------------------- |
| `Set(key, value)` | Set a header value | `response:Header():Set("Content-Type", "application/json")` |
| `Get(key)`        | Get a header value | `local ct = response:Header():Get("Content-Type")`          |

### Go-style Examples

**Basic response:**

```lua
response:WriteHeader(200)
response:Header():Set("Content-Type", "text/plain")
response:WriteString("Hello, World!")
```

**Multiple writes:**

```lua
response:WriteHeader(200)
response:Write("Line 1\n")
response:Write("Line 2\n")
response:Write("Line 3")
```

**Working with headers:**

```lua
-- Set multiple headers
response:Header():Set("Content-Type", "application/json")
response:Header():Set("X-Custom-Header", "CustomValue")
response:Header():Set("Cache-Control", "no-cache")

-- Read a header
local customValue = response:Header():Get("X-Custom-Header")

-- Write response
response:WriteHeader(200)
response:WriteString('{"status": "ok"}')
```

**Headers table with methods:**

```lua
-- Headers can be set via table or methods
response:Header():Set("X-Method-Style", "new")
response.headers["X-Table-Style"] = "old"

-- Status and body - methods only
response:WriteHeader(200)
response:WriteString("Data flows ")
response:WriteString("to network immediately")
-- Every write operation above sent data to HTTP connection in real-time
```

**Correct order example:**

```lua
-- 1. Set headers FIRST (before any writes)
response:Header():Set("Content-Type", "application/json")
response:Header():Set("X-Request-ID", "12345")

-- 2. Write status code
response:WriteHeader(200)

-- 3. Write body (can call multiple times)
response:WriteString('{"data": ')
response:WriteString('"streaming"}')
-- Data is sent to client as we write!
```

**Wrong order (will not work as expected):**

```lua
-- BAD: Writing body first
response:WriteString("Hello")  -- This auto-sends WriteHeader(200)

-- BAD: Trying to set headers after write - too late!
response:Header():Set("X-Custom", "value")  -- Headers already sent!

-- BAD: Trying to change status after write
response:WriteHeader(404)  -- Ignored! Header already sent
```

## Setting Response Properties

### Status Code

```lua
response:WriteHeader(201)  -- Created
response:WriteHeader(404)  -- Not Found
response:WriteHeader(500)  -- Internal Server Error
```

### Response Body

```lua
-- Simple text
response:WriteString("Hello, World!")

-- JSON (as string)
response:WriteString('{"message": "Success", "code": 200}')

-- Constructed dynamically
local name = "Alice"
response:WriteString('{"name": "' .. name .. '"}')
```

### Response Headers

```lua
-- Set Content-Type
response.headers["Content-Type"] = "application/json"

-- Set custom headers
response.headers["X-Custom-Header"] = "CustomValue"
response.headers["X-Request-ID"] = "12345"

-- Set cache control
response.headers["Cache-Control"] = "no-cache"
```

### Complete Example

```lua
local math = require("math")
local string = require("string")

-- Get query parameters
local min = tonumber(request.query_params["min"]) or 1
local max = tonumber(request.query_params["max"]) or 100

-- Generate random number
math.randomseed(os.time())
local random = math.random(min, max)

-- Build response
response.headers["Content-Type"] = "application/json"
response.headers["X-Generated-At"] = os.date("%Y-%m-%d %H:%M:%S")
response:WriteHeader(200)
response:WriteString('{"random": ' .. random .. ', "min": ' .. min .. ', "max": ' .. max .. '}')
```

# Available Libraries

The script handler provides access to standard libraries:

## Math Library

Mathematical functions for calculations.

```lua
local math = require("math")

-- Random numbers
local random = math.random(1, 100)
math.randomseed(os.time())

-- Rounding
local rounded = math.floor(3.7)  -- 3
local ceiling = math.ceil(3.2)   -- 4

-- Trigonometry
local sine = math.sin(1.5)
local cosine = math.cos(1.5)

-- Constants
local pi = math.pi
local huge = math.huge
```

## String Library

String manipulation and formatting.

```lua
local string = require("string")

-- Case conversion
local upper = string.upper("hello")     -- "HELLO"
local lower = string.lower("WORLD")     -- "world"

-- Substring
local sub = string.sub("Hello", 1, 3)   -- "Hel"

-- Find and replace
local pos = string.find("Hello World", "World")
local replaced = string.gsub("Hello World", "World", "Lua")

-- String length
local len = string.len("Hello")         -- 5

-- Formatting
local formatted = string.format("Value: %d", 42)
```

## Table Library

Table (array/dictionary) operations.

```lua
local table = require("table")

-- Array operations
local items = {"apple", "banana", "cherry"}
table.insert(items, "date")              -- Add to end
table.remove(items, 1)                   -- Remove first item

-- Concatenation
local joined = table.concat(items, ", ") -- "banana, cherry, date"

-- Sorting
table.sort(items)
```

## OS Library

Limited OS and time functions.

```lua
local os = require("os")

-- Time
local timestamp = os.time()
local formatted = os.date("%Y-%m-%d %H:%M:%S")
local utc = os.date("!%Y-%m-%d %H:%M:%S")  -- UTC

-- Date components
local components = os.date("*t")
-- components.year, components.month, components.day, etc.
```

## JSON Library

JSON encoding and decoding for working with JSON data.

```lua
local json = require("json")

-- Encoding (Lua to JSON)
local data = {
  name = "Alice",
  age = 30,
  active = true,
  tags = {"developer", "golang"}
}
local encoded = json.encode(data)
-- Result: {"name":"Alice","age":30,"active":true,"tags":["developer","golang"]}

-- Decoding (JSON to Lua)
local jsonString = '{"message":"hello","count":42}'
local decoded = json.decode(jsonString)
local message = decoded.message  -- "hello"
local count = decoded.count      -- 42

-- Working with request body
local requestData = json.decode(request.body)
local userId = requestData.user_id

-- Building JSON response
local responseData = {
  status = "success",
  data = {id = userId, name = "User " .. userId}
}
response.headers["Content-Type"] = "application/json"
response:WriteHeader(200)
response:WriteString(json.encode(responseData))
```

**Type mappings:**

| Lua Type               | JSON Type |
| ---------------------- | --------- |
| `nil`                  | `null`    |
| `number`               | `number`  |
| `string`               | `string`  |
| `boolean`              | `boolean` |
| `table` (string keys)  | `object`  |
| `table` (numeric keys) | `array`   |

**Error handling:**

```lua
local json = require("json")

-- Decode with error handling
local success, result = pcall(json.decode, request.body)
if not success then
  response:WriteHeader(400)
  response:WriteString('{"error": "Invalid JSON"}')
  return
end

-- Use decoded data
response:WriteHeader(200)
response:WriteString(json.encode({received = result}))
```

# Complete Examples

## Simple API Endpoint

```yaml
scripts:
  - path: /api/health
    method: GET
    script: |
      response.headers["Content-Type"] = "application/json"
      response:WriteHeader(200)
      response:WriteString('{"status": "healthy", "timestamp": "' .. os.date("%Y-%m-%d %H:%M:%S") .. '"}')
```

## Dynamic User API

```yaml
scripts:
  - path: /api/users/{id}
    method: GET
    script: |
      local userId = request.path_params["id"] or "unknown"

      response.headers["Content-Type"] = "application/json"
      response:WriteHeader(200)
      response:WriteString('{' ..
        '"id": "' .. userId .. '",' ..
        '"name": "User ' .. userId .. '",' ..
        '"email": "user' .. userId .. '@example.com"' ..
      '}')
```

## Calculator API

```yaml
scripts:
  - path: /api/calculate
    method: POST
    script: |
      local math = require("math")

      -- Get operation from query params
      local op = request.query_params["operation"]

      response.headers["Content-Type"] = "application/json"

      if op == "random" then
        local min = tonumber(request.query_params["min"]) or 1
        local max = tonumber(request.query_params["max"]) or 100
        math.randomseed(os.time())
        local result = math.random(min, max)

        response:WriteHeader(200)
        response:WriteString('{"result": ' .. result .. '}')
      elseif op == "sqrt" then
        local value = tonumber(request.query_params["value"]) or 0
        local result = math.sqrt(value)

        response:WriteHeader(200)
        response:WriteString('{"result": ' .. result .. '}')
      else
        response:WriteHeader(400)
        response:WriteString('{"error": "Unknown operation"}')
      end
```

## Request Echo Service

```yaml
scripts:
  - path: /api/echo
    script: |
      response.headers["Content-Type"] = "application/json"
      response:WriteHeader(200)
      response:WriteString('{' ..
        '"method": "' .. request.method .. '",' ..
        '"path": "' .. request.path .. '",' ..
        '"query": "' .. request.query .. '",' ..
        '"body": "' .. request.body .. '",' ..
        '"host": "' .. request.host .. '"' ..
      '}')
```

## Conditional Response Based on Headers

```yaml
scripts:
  - path: /api/data
    method: GET
    script: |
      local authHeader = request.headers["Authorization"]

      response.headers["Content-Type"] = "application/json"

      if authHeader and string.find(authHeader, "Bearer ") then
        response:WriteHeader(200)
        response:WriteString('{"data": "Secret information", "authorized": true}')
      else
        response.headers["WWW-Authenticate"] = 'Bearer realm="API"'
        response:WriteHeader(401)
        response:WriteString('{"error": "Unauthorized", "authorized": false}')
      end
```

## JSON API with Request Body Processing

```yaml
scripts:
  - path: /api/users
    method: POST
    script: |
      local json = require("json")

      -- Parse JSON request body
      local success, userData = pcall(json.decode, request.body)
      if not success then
        response.headers["Content-Type"] = "application/json"
        response:WriteHeader(400)
        response:WriteString(json.encode({
          error = "Invalid JSON in request body"
        }))
        return
      end

      -- Validate required fields
      if not userData.name or not userData.email then
        response.headers["Content-Type"] = "application/json"
        response:WriteHeader(400)
        response:WriteString(json.encode({
          error = "Missing required fields: name and email"
        }))
        return
      end

      -- Create response with JSON
      local responseData = {
        id = os.time(),
        name = userData.name,
        email = userData.email,
        created_at = os.date("%Y-%m-%d %H:%M:%S"),
        status = "active"
      }

      response.headers["Content-Type"] = "application/json"
      response:WriteHeader(201)
      response:WriteString(json.encode(responseData))
```

# CORS Headers

CORS headers are automatically added to all script responses. You can override them by setting custom header values in your script:

```lua
-- CORS headers are added automatically, but you can override them
response.headers["Access-Control-Allow-Origin"] = "https://example.com"
response.headers["Access-Control-Allow-Methods"] = "GET, POST"
response.headers["Access-Control-Allow-Headers"] = "Content-Type, Authorization"
```

# Error Handling

If your script encounters an error, the handler will return a 500 Internal Server Error response automatically. Common errors include:

- **Script not defined**: Neither `script` nor `file` is specified
- **Both script and file defined**: Only one should be specified
- **File not found**: The specified script file doesn't exist
- **Script syntax error**: Invalid script syntax in your script
- **Script runtime error**: Error during script execution (e.g., accessing nil values)

**Best practices:**

```lua
-- Check for nil values before accessing
if request.query_params["id"] then
    local id = request.query_params["id"]
    -- Use id safely
else
    response:WriteHeader(400)
    response:WriteString('{"error": "Missing id parameter"}')
end

-- Use pcall for error handling
local success, result = pcall(function()
    -- Your code that might error
    return someFunction()
end)

if not success then
    response:WriteHeader(500)
    response:WriteString('{"error": "Internal error"}')
end
```

# Tips and Best Practices

1. **Keep scripts simple**: scripts are executed for each request, so keep logic lightweight
2. **Use file-based scripts for complex logic**: Easier to test and maintain
3. **Validate input**: Always validate query parameters and headers before using them
4. **Set Content-Type**: Always set the appropriate Content-Type header for your response
5. **Handle errors gracefully**: Check for nil values and provide meaningful error messages
6. **Use libraries**: Leverage math, string, and table libraries for common operations
7. **Test scripts separately**: scripts can be tested independently before integration
8. **Escape JSON strings**: Be careful with quotes when building JSON strings dynamically
9. **⚠️ CRITICAL: Follow HTTP order**: Set headers → WriteHeader → Write body. Headers cannot be modified after first write!
10. **Use methods for status/body**: Cannot assign to `response.status` or `response.body` directly - use methods
11. **Cannot read status/body**: `response.status` and `response.body` return `nil` if read
12. **Streaming-ready**: Every write goes directly to network - perfect for streaming responses
13. **Performance**: Zero buffering means minimal memory usage and immediate data transmission

# Comparison with Mock Handler

| Feature             | Script Handler                                | Mock Handler                         |
| ------------------- | --------------------------------------------- | ------------------------------------ |
| **Flexibility**     | Full control with script code                 | Pre-defined response types           |
| **Dynamic content** | Yes, fully programmable                       | Limited to fake data schemas         |
| **Request access**  | Full access to all request properties         | No request access                    |
| **Complex logic**   | Yes, use any script code                      | No logic, configuration-only         |
| **Learning curve**  | Requires scripting knowledge                  | Simple YAML configuration            |
| **Use case**        | Custom business logic, complex API simulation | Simple mocking, fake data generation |

Choose the **Script Handler** when you need:

- Custom business logic
- Request-dependent responses
- Complex data transformations
- Conditional responses based on request data

Choose the **Mock Handler** when you need:

- Simple static responses
- Quick fake data generation
- No custom logic required
