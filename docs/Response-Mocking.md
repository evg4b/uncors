Mocks return a predefined response for matching requests without contacting
the upstream server. Use them to fake endpoints that don't exist yet, to force
error responses, or to simulate slow responses.

A mock matches on the request path and, optionally, the method, query
parameters, and headers. Matching uses
[gorilla/mux](https://github.com/gorilla/mux#matching-routes) routes.

```yaml
mappings:
  - from: http://api.local:3000
    to: https://api.example.com
    mocks:
      - path: /example-endpoint
        method: POST
        queries:
          param1: value1
          param2: value2
        headers:
          Content-Type: application/json
        response:
          code: 201
          headers:
            Content-Type: application/json
          delay: 10s
          raw: '{ "ok": true }'
```

## Request matching

| Option    | Required | Description                                                             |
| --------- | -------- | ----------------------------------------------------------------------- |
| `path`    | Yes      | URL path to match. Must start with `/`.                                 |
| `method`  | No       | HTTP method in upper case, for example `GET` or `POST`. Default: any.   |
| `queries` | No       | Query parameters that must be present with these values.               |
| `headers` | No       | Request headers that must be present with these values.                |

### Path

The path must match the whole request path. `/api/users` matches `/api/users`
but not `/api/users/` or `/api/users/1`.

A segment in braces matches any value in that position:

```yaml
path: /api/users              # exact path
path: /users/{id}             # /users/123, /users/abc, ...
path: /posts/{postId}/comments/{commentId}
```

gorilla/mux also accepts a regular expression after the name, for example
`/users/{id:[0-9]+}`.

### Method

```yaml
method: DELETE
```

Without `method`, the mock answers every method. Allowed values are `GET`,
`HEAD`, `POST`, `PUT`, `PATCH`, `DELETE`, `CONNECT`, `OPTIONS`, and `TRACE`.

### Query parameters and headers

```yaml
queries:
  page: "1"
headers:
  Authorization: Bearer token123
```

Every listed parameter and header must be present with the given value. Other
parameters and headers in the request are ignored.

### Matching order

Mocks that filter on method, query parameters, or headers are checked before
mocks that match on path alone, whatever their order in the file. Among mocks
of the same kind, the first one in the file wins. This lets you combine a
specific mock with a catch-all for the same path:

```yaml
mocks:
  - path: /api/users/{id}
    response:
      code: 200
      raw: '{"id": "123"}'
  - path: /api/users/{id}
    method: DELETE
    response:
      code: 403
      raw: '{"error": "forbidden"}'
```

Here `DELETE /api/users/1` returns 403 and any other method returns 200.

Mocks are checked before [scripts](Script-Handler) and
[rewrites](Request-Rewriting). [Static directories](Static-File-Serving) are
checked before mocks, so a mock whose path falls under a static `path` prefix is
never reached.

## Response

| Option    | Type    | Required | Description                                                  |
| --------- | ------- | -------- | ------------------------------------------------------------ |
| `code`    | integer | Yes      | HTTP status code, 100 to 599.                                |
| `headers` | object  | No       | Response headers.                                            |
| `delay`   | string  | No       | Wait this long before responding, for example `500ms`.       |
| `raw`     | string  | One of   | Response body as text.                                       |
| `file`    | string  | One of   | Path to a file whose content is the response body.          |

Set exactly one of `raw` and `file`.

### Body from `raw`

```yaml
response:
  code: 200
  headers:
    Content-Type: application/json
  raw: '{"message": "Success", "id": 123}'
```

If you don't set `Content-Type`, UNCORS guesses it from the first bytes of the
body. JSON is not detected and is sent as `text/plain`, so set the header
yourself for JSON.

### Body from `file`

```yaml
response:
  code: 200
  file: ./mocks/users-response.json
```

The path is relative to the directory UNCORS was started from. UNCORS checks
that the file exists at startup and reads it on every request, so edits to the
file show up without a restart. `Content-Type` is taken from the file
extension.

> [!NOTE]
> File responses are currently always sent with status 200, whatever `code`
> says. Use `raw`, or a [script](Script-Handler), when you need another status
> code.

### Headers

```yaml
response:
  code: 200
  headers:
    Content-Type: application/json
    X-Custom-Header: value
```

UNCORS adds its CORS headers to mock responses. Headers you list here are set
after them, so you can override any of them.

### Delay

`delay` simulates a slow endpoint. It uses Go duration syntax: a number
followed by a unit, with several parts written together without spaces.

| Unit       | Meaning      |
| ---------- | ------------ |
| `ns`       | Nanoseconds  |
| `us`, `µs` | Microseconds |
| `ms`       | Milliseconds |
| `s`        | Seconds      |
| `m`        | Minutes      |
| `h`        | Hours        |

```yaml
delay: 500ms
delay: 1m30s
delay: 2s500ms
```

If the client disconnects during the delay, UNCORS stops waiting.

## Dynamic responses

Mocks always return the same response. To build a response from the request,
use the [Script Handler](Script-Handler).
