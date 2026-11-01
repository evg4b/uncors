Rewrites change the request path, and optionally the upstream host, before a
request is proxied. Use them when the URLs your client calls don't match the
paths the server expects, or to send some paths to a different backend.

```yaml
mappings:
  - from: http://localhost:3000
    to: https://api.example.com
    rewrites:
      - from: /api/{resource}
        to: /api/v1/{resource}
        host: https://external-api.example.com
```

| Option | Type   | Required | Description                                                         |
| ------ | ------ | -------- | ------------------------------------------------------------------- |
| `from` | string | Yes      | Path to match. Can contain `{name}` segments.                       |
| `to`   | string | Yes      | New path. Can use the `{name}` values captured by `from`.           |
| `host` | string | No       | Upstream host, with optional scheme and port, for matched requests. |

## How matching works

A rule matches a path that equals `from` or starts with `from` followed by
`/`. The whole path is then replaced with `to`, so anything after the matched
part is dropped. The query string is kept.

```yaml
rewrites:
  - from: /api/{resource}
    to: /api/v1/{resource}/list
```

| Incoming request    | Upstream request            |
| ------------------- | --------------------------- |
| `/api/users`        | `/api/v1/users/list`        |
| `/api/posts/42`     | `/api/v1/posts/list`        |
| `/api/posts?page=2` | `/api/v1/posts/list?page=2` |

A `{name}` segment matches one path segment. Capture each segment you need:

```yaml
rewrites:
  - from: /users/{userId}/posts/{postId}
    to: /api/users/{userId}/content/posts/{postId}
```

To keep the rest of the path
whatever its length, end `from` with `{name:.*}`:

```yaml
rewrites:
  - from: /files/{rest:.*}
    to: /v2/files/{rest}
```

| Incoming request              | Upstream request                 |
| ----------------------------- | -------------------------------- |
| `/files/docs/2026/report.pdf` | `/v2/files/docs/2026/report.pdf` |
| `/files/logo.png`             | `/v2/files/logo.png`             |

A rewritten request always goes to the upstream server, even if the new path
matches a mock or a script. Caching and HAR recording still apply, using the
rewritten path. Static directories, mocks, and scripts are checked before
rewrites, so they win when paths overlap:

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
- `GET /old-api/users` is rewritten, proxied, and cached under
  `/v2/api/users`.

## Changing the upstream host

`host` sends matched requests to another server than the mapping's `to`:

```yaml
mappings:
  - from: http://localhost
    to: https://primary-api.example.com
    rewrites:
      - from: /auth/{endpoint}
        to: /v1/{endpoint}
        host: https://auth-service.example.com
      - from: /payment/{endpoint}
        to: /v2/{endpoint}
        host: https://payment-service.example.com
```

| Request                 | Upstream request                                      |
| ----------------------- | ----------------------------------------------------- |
| `GET /auth/login`       | `GET https://auth-service.example.com/v1/login`       |
| `POST /payment/process` | `POST https://payment-service.example.com/v2/process` |
| `GET /users`            | `GET https://primary-api.example.com/users`           |

> [!NOTE]
> A `host` without a scheme uses the scheme of the incoming request, not the
> scheme of `to`. From an `http://` source, `host: auth-service.example.com`
> is called over plain HTTP.
