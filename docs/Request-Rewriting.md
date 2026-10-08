Rewrites change the request path, and optionally the upstream host, before a
request is proxied. Use them when the local URLs your client calls don't match
the paths the server expects, or to send some paths to a different backend.

```yaml
mappings:
  - from: http://localhost:3000
    to: https://api.example.com
    rewrites:
      - from: /api/{resource}
        to: /api/v1/{resource}
        host: external-api.example.com
```

## Options

| Option | Type   | Required | Description                                                     |
| ------ | ------ | -------- | --------------------------------------------------------------- |
| `from` | string | Yes      | Path to match. Can contain `{name}` segments.                   |
| `to`   | string | Yes      | New path. Can reuse the `{name}` values captured by `from`.     |
| `host` | string | No       | Upstream host (and optional port) for requests matched by this rule. |

## How matching works

A rule matches requests whose path equals `from` or starts with `from`
followed by `/`. The whole request path is then replaced with `to`, after
substituting placeholders. Anything after the matched part is dropped, and so
is the query string.

```yaml
rewrites:
  - from: /api/{resource}
    to: /api/v1/{resource}/list
```

| Incoming request | Upstream request        |
| ---------------- | ----------------------- |
| `/api/users`     | `/api/v1/users/list`    |
| `/api/posts`     | `/api/v1/posts/list`    |
| `/api/posts/42`  | `/api/v1/posts/list`    |

A `{name}` segment matches one path segment. To keep more of the path,
capture each segment you need:

```yaml
rewrites:
  - from: /users/{userId}/posts/{postId}
    to: /api/users/{userId}/content/posts/{postId}
```

A rewritten request always goes to the upstream server. It does not reach
mocks or scripts, even if the new path matches one. Caching and HAR recording
still apply, and the cache uses the rewritten path. Rewrites are checked after
static directories, mocks, and scripts, so those win when their paths overlap
with a rewrite.

## Changing the upstream host

`host` sends matched requests to a different server than the mapping's `to`:

```yaml
mappings:
  - from: http://localhost
    to: https://primary-api.example.com
    rewrites:
      - from: /auth/{endpoint}
        to: /v1/{endpoint}
        host: auth-service.example.com
      - from: /payment/{endpoint}
        to: /v2/{endpoint}
        host: payment-service.example.com
```

| Request                 | Upstream request                                   |
| ----------------------- | -------------------------------------------------- |
| `GET /auth/login`       | `GET http://auth-service.example.com/v1/login`     |
| `POST /payment/process` | `POST http://payment-service.example.com/v2/process` |
| `GET /users`            | `GET https://primary-api.example.com/users` (no rule matched) |

The request to the rewritten host uses the scheme of the incoming request, not
the scheme of `to`. A scheme written in `host` is ignored. In the example
above the local side is `http`, so the auth and payment services are called
over plain HTTP. Use an `https://` source if the rewritten host needs HTTPS.

## Combining rewrites with other features

```yaml
mappings:
  - from: http://localhost:3000
    to: https://api.example.com
    rewrites:
      - from: /old-api/{resource}
        to: /v2/api/{resource}
    mocks:
      - path: /v2/api/health
        response:
          code: 200
          raw: '{"status": "healthy"}'
    cache:
      - /v2/api/users
```

- `GET /v2/api/health` returns the mock.
- `GET /old-api/health` is rewritten to `/v2/api/health` and proxied; the mock
  does not answer it.
- `GET /old-api/users` is rewritten to `/v2/api/users`, proxied, and cached
  under the rewritten path.
