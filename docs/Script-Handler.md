Scripts build responses with Lua. Use them when a fixed [mock](Response-Mocking)
is not enough: when the response depends on the request, needs some logic, or
has to call an external tool. Scripts run on
[gopher-lua](https://github.com/yuin/gopher-lua), a Lua 5.1 implementation.

```yaml
mappings:
  - from: http://api.local:3000
    to: https://api.example.com
    scripts:
      - path: /api/greeting
        method: GET
        script: |
          local name = request.query_params["name"] or "World"
          response.headers["Content-Type"] = "application/json"
          response:WriteHeader(200)
          response:WriteString('{"message": "Hello, ' .. name .. '"}')
```

## Contents

- [Request matching](#request-matching)
- [Inline and file scripts](#inline-and-file-scripts)
- [The request object](#the-request-object)
- [The response object](#the-response-object)
- [Libraries](#libraries)
- [Examples](#examples)
- [Errors](#errors)
- [Scripts or mocks](#scripts-or-mocks)

## Request matching

Scripts match requests the same way mocks do:

| Option    | Required | Description                                                           |
| --------- | -------- | --------------------------------------------------------------------- |
| `path`    | Yes      | URL path to match, starting with `/`. `{name}` segments match any value. |
| `method`  | No       | HTTP method in upper case. Default: any.                              |
| `queries` | No       | Query parameters that must be present with these values.             |
| `headers` | No       | Request headers that must be present with these values.              |

The path must match the whole request path. Scripts with a method, query, or
header filter are checked before scripts that match on path alone. Mocks are
checked before scripts, and static directories before both. See
[Response Mocking](Response-Mocking#request-matching) for the details, which
are the same.

## Inline and file scripts

Each script has exactly one of `script` (inline code) or `file` (a path to a
`.lua` file):

```yaml
scripts:
  - path: /api/greeting
    script: |
      response:WriteString("Hello")

  - path: /api/calculate
    method: POST
    file: ./scripts/calculator.lua
```

`./scripts/calculator.lua`:

```lua
local json = require("json")
local input = json.decode(request.body)

response.headers["Content-Type"] = "application/json"
response:WriteHeader(200)
response:WriteString(json.encode({result = input.a + input.b}))
```

File paths are relative to the directory you start UNCORS from. UNCORS checks
that the file exists at startup and reads it again on every request, so you can
edit a script without restarting. Every request runs in a fresh Lua state;
nothing is kept between requests.

## The request object

The global `request` table describes the incoming request.

| Field          | Type   | Description                                       | Example                                 |
| -------------- | ------ | ------------------------------------------------- | --------------------------------------- |
| `method`       | string | HTTP method                                       | `"GET"`                                 |
| `url`          | string | Request URL                                       | `"http://localhost:3000/api/users?id=1"` |
| `path`         | string | URL path                                          | `"/api/users"`                          |
| `query`        | string | Raw query string                                  | `"id=1&name=test"`                      |
| `host`         | string | `Host` header                                     | `"localhost:3000"`                      |
| `remote_addr`  | string | Client address and port                           | `"127.0.0.1:52345"`                     |
| `body`         | string | Request body, empty string if there is none       | `'{"data": "value"}'`                   |
| `headers`      | table  | Request headers                                   | `request.headers["Content-Type"]`       |
| `query_params` | table  | Parsed query parameters                           | `request.query_params["id"]`            |
| `path_params`  | table  | Values of the `{name}` segments in `path`         | `request.path_params["id"]`             |

Header names use Go's canonical form, so read `request.headers["Content-Type"]`,
not `request.headers["content-type"]`.

In `headers` and `query_params`, a name that appears once maps to a string. A
name that appears several times maps to an array of strings. Check the type if
a client might repeat a value:

```lua
local tags = request.query_params["tag"]
if type(tags) == "string" then
  tags = {tags}
end
```

Path parameters come from the `{name}` segments of the script's `path`:

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

## The response object

The global `response` object writes directly to the HTTP response, following
the rules of Go's `http.ResponseWriter`:

1. Set headers.
2. Call `response:WriteHeader(code)` once to send the status line and headers.
3. Write the body with `response:Write` or `response:WriteString`, as many
   times as you need.

Once the status line is sent, later header changes and later `WriteHeader`
calls are ignored. If you write body data without calling `WriteHeader` first,
status 200 is sent automatically. If the script writes nothing, the client gets
an empty 200 response.

| Member                       | Description                                       |
| ---------------------------- | ------------------------------------------------- |
| `response.headers[name]`     | Read or set a response header                     |
| `response:Header()`          | Returns the headers object, with `Set` and `Get`  |
| `response:WriteHeader(code)` | Send the status code and headers                  |
| `response:Write(data)`       | Append data to the body                           |
| `response:WriteString(str)`  | Append a string to the body (same as `Write`)     |

`Write` and `WriteString` return the number of bytes written and an error
message, or `nil` if there was no error.

Headers can be set either way:

```lua
response.headers["Content-Type"] = "application/json"
response:Header():Set("X-Request-ID", "12345")
local contentType = response:Header():Get("Content-Type")
```

There is no `response.status` or `response.body` field. Assigning to them does
nothing and reading them returns `nil`; use `WriteHeader` and `Write` instead.

The order matters:

```lua
-- Correct
response:Header():Set("Content-Type", "application/json")
response:WriteHeader(201)
response:WriteString('{"created": true}')

-- Wrong: the header is set after the status line was sent and is lost
response:WriteHeader(201)
response:Header():Set("Content-Type", "application/json")
```

UNCORS adds its CORS headers before the script runs, so a script can replace
them:

```lua
response.headers["Access-Control-Allow-Origin"] = "https://example.com"
```

## Libraries

All standard gopher-lua libraries are loaded, including `string`, `table`,
`math`, `os`, and `io`. They are available as globals, and `require("math")`
and similar calls also work.

A JSON module from [gopher-json](https://github.com/layeh/gopher-json) is
available through `require("json")`:

```lua
local json = require("json")

local encoded = json.encode({name = "Alice", tags = {"dev", "go"}})
local decoded = json.decode('{"message": "hello", "count": 42}')
-- decoded.message == "hello", decoded.count == 42
```

| Lua value             | JSON value |
| --------------------- | ---------- |
| `nil`                 | `null`     |
| number                | number     |
| string                | string     |
| boolean               | boolean    |
| table with string keys | object    |
| table with array keys  | array     |

On invalid input, `json.decode` does not raise an error. It returns `nil` and
an error message, so check the result before using it:

```lua
local json = require("json")

local data, err = json.decode(request.body)
if data == nil then
  response:WriteHeader(400)
  response:WriteString(json.encode({error = "invalid JSON: " .. tostring(err)}))
  return
end
```

> [!WARNING]
> Scripts are not sandboxed. `os.execute`, `io.popen`, and file functions in
> `io` and `os` run with the permissions of the UNCORS process. Only run
> scripts you trust.

## Examples

### Health endpoint

```yaml
scripts:
  - path: /api/health
    method: GET
    script: |
      response.headers["Content-Type"] = "application/json"
      response:WriteHeader(200)
      response:WriteString('{"status": "healthy", "time": "' .. os.date("%Y-%m-%d %H:%M:%S") .. '"}')
```

### Random number in a range

```yaml
scripts:
  - path: /api/random
    script: |
      local min = tonumber(request.query_params["min"]) or 1
      local max = tonumber(request.query_params["max"]) or 100
      math.randomseed(os.time())

      response.headers["Content-Type"] = "application/json"
      response:WriteHeader(200)
      response:WriteString('{"random": ' .. math.random(min, max) .. '}')
```

### Response that depends on a header

```yaml
scripts:
  - path: /api/data
    method: GET
    script: |
      local auth = request.headers["Authorization"]
      response.headers["Content-Type"] = "application/json"

      if auth and string.find(auth, "Bearer ", 1, true) then
        response:WriteHeader(200)
        response:WriteString('{"data": "secret", "authorized": true}')
      else
        response.headers["WWW-Authenticate"] = 'Bearer realm="API"'
        response:WriteHeader(401)
        response:WriteString('{"error": "unauthorized"}')
      end
```

### Validating a JSON body

```yaml
scripts:
  - path: /api/users
    method: POST
    script: |
      local json = require("json")
      response.headers["Content-Type"] = "application/json"

      local user = json.decode(request.body)
      if type(user) ~= "table" then
        response:WriteHeader(400)
        response:WriteString(json.encode({error = "invalid JSON"}))
        return
      end

      if not user.name or not user.email then
        response:WriteHeader(400)
        response:WriteString(json.encode({error = "name and email are required"}))
        return
      end

      response:WriteHeader(201)
      response:WriteString(json.encode({
        id = os.time(),
        name = user.name,
        email = user.email,
        created_at = os.date("%Y-%m-%d %H:%M:%S"),
      }))
```

### Echo

`json.encode` takes care of quoting, which string concatenation does not:

```yaml
scripts:
  - path: /api/echo
    script: |
      local json = require("json")
      response.headers["Content-Type"] = "application/json"
      response:WriteHeader(200)
      response:WriteString(json.encode({
        method = request.method,
        path = request.path,
        query = request.query,
        body = request.body,
        host = request.host,
      }))
```

## Errors

Configuration errors are reported at startup: a script entry with neither
`script` nor `file`, with both, or with a `file` that doesn't exist.

If a script raises an error at runtime (a syntax error, a call on `nil`, an
`error(...)` call), UNCORS prints the error to the console and answers with
status 500, as long as the script had not already sent the status line. Use
`pcall` around code that can fail when you want to choose the response
yourself.

## Scripts or mocks

| Need                                   | Use    |
| -------------------------------------- | ------ |
| The same response every time           | Mock   |
| A response body stored in a file       | Mock   |
| A response that depends on the request | Script |
| Logic, validation, or calling a tool   | Script |
| A simulated delay                      | Mock (`delay`) |

Mocks need no code and are easier to read. Scripts can do anything Lua can.
