Request rewriting allows you to transform request paths and hosts before they
are forwarded to the upstream server. This is useful for:

 - Adapting client URLs to match server API structure
 - Routing requests to different backend services
 - Versioning API endpoints
 - Migrating between API versions

**Configuration:**

Add a `rewrites` section to your mapping configuration:

```yaml
mappings:
  - from: http://localhost:3000
    to: https://api.example.com
    rewrites:
      - from: /api
        to: /api/v1
        host: external-api.example.com
```

## Configuration Properties

| Property | Type   | Required | Description                                                            |
| -------- | ------ | -------- | ---------------------------------------------------------------------- |
| `from`   | string | Yes      | Path to match, with `{name}` **path variables**                        |
| `to`     | string | Yes      | Replacement path; may use the variables `from` captured                |
| `host`   | string | No       | Send this rule's requests to another host, optionally with a scheme    |

## Path Variables

Capture parts of the URL with `{name}` and reference them in the target path. A
path variable matches **one path segment** and does not cross a `/`.

> [!NOTE]
> uncors uses three different pattern syntaxes, one per job: **path variables**
> (`{name}`) in rewrites, **host placeholders** (`{name}`) in mapping hosts, and
> **glob patterns** (`**`) in `cache`. A `{name}` in `to` that `from` does not
> capture is a configuration error.

```yaml
mappings:
  - from: http://localhost:3000
    to: https://api.example.com
    rewrites:
      - from: /api/{resource}
        to: /api/v1/{resource}/list
```

The `{resource}` placeholder captures part of the incoming path and inserts it
into the rewritten path:

| Incoming Request | Rewritten Request       |
| ---------------- | ----------------------- |
| `/api/users`     | `/api/v1/users/list`    |
| `/api/posts`     | `/api/v1/posts/list`    |
| `/api/products`  | `/api/v1/products/list` |

## Examples

### API Versioning

Redirect old API paths to new versioned endpoints:

```yaml
mappings:
  - from: http://localhost
    to: https://api.example.com
    rewrites:
      - from: /v1/{endpoint}
        to: /api/v2/{endpoint}
```

### Multiple Path Segments

Capture multiple URL segments:

```yaml
mappings:
  - from: http://localhost
    to: https://api.example.com
    rewrites:
      - from: /users/{userId}/posts/{postId}
        to: /api/users/{userId}/content/posts/{postId}
```

### Host Rewriting

Route specific paths to different backend services:

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

> [!IMPORTANT]
> Include the scheme in `host` when the other service speaks a different one.
> Without it, the incoming request's scheme is kept — so a rule reached over
> `http://` would go out over `http://` too, and cookies coming back would not be
> marked secure.

**Request flow:**

 - `GET /auth/login` → `GET https://auth-service.example.com/v1/login`
 - `POST /payment/process` →
   `POST https://payment-service.example.com/v2/process`
 - `GET /users` → `GET https://primary-api.example.com/users` (no rewrite
   applied)

### Combining Rewrites with Other Features

Rewrites, mocks, and caching can be used together in a single mapping. A
rewritten request re-enters the mapping's routes, so `/old-api/health` below is
rewritten to `/v2/api/health` and then answered by the mock at that path:

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
      - /v2/api/users/**
```

Rules that keep sending a request back into each other are stopped after eight
rounds, and the request is answered with `508 Loop Detected`.
