Mocks return a predefined response for matching requests without contacting
the upstream server. Use them to fake endpoints that don't exist yet, to force
error responses, or to simulate slow responses.

```yaml
mappings:
  - from: http://api.local:3000
    to: https://api.example.com
    mocks:
      - path: /example-endpoint
        method: POST
        queries:
          param1: value1
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

| Option    | Required | Description                                                           |
| --------- | -------- | --------------------------------------------------------------------- |
| `path`    | Yes      | URL path to match, starting with `/`.                                 |
| `method`  | No       | HTTP method in upper case, for example `GET` or `POST`. Default: any. |
| `queries` | No       | Query parameters that must be present with these values.              |
| `headers` | No       | Request headers that must be present with these values.               |

The path must match the whole request path: `/api/users` does not match
`/api/users/` or `/api/users/1`. A segment in braces matches any value, and
can take a regular expression
([gorilla/mux](https://github.com/gorilla/mux#matching-routes) syntax):

```yaml
path: /users/{id}
path: /users/{id:[0-9]+}
path: /posts/{postId}/comments/{commentId}
```

Every listed query parameter and header must be present with the given value;
others are ignored.

Mocks with a `method`, `queries`, or `headers` filter are checked before mocks
that match on path alone, whatever their order in the file. Otherwise the
first match wins. So a specific mock and a catch-all can share a path:

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

> [!NOTE]
> Without `method`, a mock does not answer `OPTIONS`; UNCORS answers the
> preflight itself (see
> [OPTIONS request handling](Configuration#options-request-handling)).
> [Static directories](Static-File-Serving) are checked before mocks, so a mock
> under a static `path` is never reached.

## Response

| Option    | Type    | Required | Description                                             |
| --------- | ------- | -------- | ------------------------------------------------------- |
| `code`    | integer | Yes      | HTTP status code, 200 to 599.                           |
| `headers` | object  | No       | Response headers.                                       |
| `delay`   | string  | No       | Wait before responding, for example `500ms` or `1m30s`. |
| `raw`     | string  | One of   | Response body as text.                                  |
| `file`    | string  | One of   | Path to a file whose content is the response body.      |

Set exactly one of `raw` and `file`. `204` and `304` need neither, and never
send a body.

```yaml
response:
  code: 200
  headers:
    Content-Type: application/json
  raw: '{"message": "Success", "id": 123}'
```

```yaml
response:
  code: 404
  file: ./mocks/not-found.json
```

A `file` path is relative to the directory UNCORS was started from. The file
is read on every request, so edits show up without a restart. Its
`Content-Type` comes from the extension.

Without a `Content-Type` header, UNCORS guesses the type from the body. JSON
is not detected and is sent as `text/plain`, so set the header for JSON.

Mock responses get the usual CORS headers. Headers you list under `headers`
are set after them and can override them.

`delay` uses Go duration syntax without spaces, such as `500ms`, `2s`, or
`1m30s`. If the client disconnects during the delay, UNCORS stops waiting.

> [!NOTE]
> A `file` mock with `code: 200` also answers conditional and range requests
> with `304` or `206`.

To build a response from the request, use the [Script Handler](Script-Handler).
