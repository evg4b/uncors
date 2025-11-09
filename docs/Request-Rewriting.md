Request rewriting allows you to transform request paths and hosts before they're forwarded to the upstream server. This is useful for:

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

| Property | Type   | Required | Description                                   |
| -------- | ------ | -------- | --------------------------------------------- |
| `from`   | string | Yes      | Path pattern to match (supports wildcards)    |
| `to`     | string | Yes      | Replacement path pattern (supports wildcards) |
| `host`   | string | No       | Override upstream host for this rewrite rule  |

## Wildcard Support

Capture parts of the URL using `{variable}` syntax and reference them in the target path:

**Example:**

```yaml
mappings:
  - from: http://localhost:3000
    to: https://api.example.com
    rewrites:
      - from: /api/{resource}
        to: /api/v1/{resource}/list
```

**How it works:**

The `{resource}` placeholder captures part of the incoming path and inserts it into the rewritten path.

**Request transformations:**

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
        host: auth-service.example.com
      - from: /payment/{endpoint}
        to: /v2/{endpoint}
        host: payment-service.example.com
```

**Request flow:**

- `GET /auth/login` → `GET https://auth-service.example.com/v1/login`
- `POST /payment/process` → `POST https://payment-service.example.com/v2/process`
- `GET /users` → `GET https://primary-api.example.com/users` (no rewrite)

### Combining Rewrites with Other Features

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
