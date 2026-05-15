Configure UNCORS to serve static files from local directories. This feature is useful for:

- Running local Single-Page Applications (SPAs)
- Overriding specific assets from remote servers
- Testing UI changes without deploying
- Serving custom resources during development

Static file configuration is defined per mapping:

```yaml
mappings:
  - from: ...
    to: ...
    statics:
      - path: /assets
        dir: ~/project/assets
        index: index.html
      - path: /static
        dir: ~/project/data
```

## Configuration Properties

| Property | Type   | Required | Description                                                     |
| -------- | ------ | -------- | --------------------------------------------------------------- |
| `path`   | string | Yes      | URL path prefix for serving files (wildcards not supported)     |
| `dir`    | string | Yes      | Local directory path containing files to serve                  |
| `index`  | string | No       | Fallback file when requested file not found (relative to `dir`) |

**Request handling behavior:**

- Requests matching the `path` prefix are served from the local `dir`
- If a file exists locally, it is served immediately
- If a file does not exist and `index` is set, the index file is served (SPA mode)
- If a file does not exist and `index` is not set, the request is forwarded upstream (proxy mode)

## SPA Mode

Single-Page Application (SPA) mode serves a fallback file for all unmatched requests. This is essential for client-side routing frameworks like React Router, Vue Router, or Angular Router.

**How it works:**

1. Requests matching the `path` prefix are checked against local files
2. If a file exists (e.g., `/app/bundle.js`), it is served directly
3. If no file exists (e.g., `/app/users/123`), the `index` file is returned
4. The SPA's JavaScript router handles the URL and renders the appropriate view

**Configuration:**

```yaml
mappings:
  - from: ...
    to: ...
    statics:
      - path: /app
        dir: ~/project/dist
        index: index.html
```

**Use cases:**

- Serving a built React, Vue, or Angular application
- Local development with client-side routing
- Testing production builds locally

## Proxy Mode

Proxy mode serves local files when they exist, but forwards unmatched requests to the upstream server or mock handlers.

**How it works:**

1. Requests matching the `path` prefix are checked against local files
2. If a file exists locally, it is served from `dir`
3. If no file exists, the request passes to the next handler (mock or upstream)

**Configuration:**

```yaml
mappings:
  - from: http://localhost
    to: https://api.example.com
    statics:
      - path: /assets
        dir: ~/project/dist
```

**Use cases:**

- Overriding specific assets (stylesheets, images, JavaScript)
- Testing local modifications without deploying
- Mixing local and remote resources

## Examples

### Serving Multiple Static Directories

```yaml
mappings:
  - from: http://localhost
    to: https://example.com
    statics:
      - path: /app
        dir: ~/project/dist
        index: index.html
      - path: /docs
        dir: ~/project/documentation
      - path: /images
        dir: ~/project/assets/img
```

### SPA with API Proxying

```yaml
mappings:
  - from: http://localhost:3000
    to: https://api.example.com
    statics:
      - path: /
        dir: ~/my-app/build
        index: index.html
    mocks:
      - path: /api/health
        response:
          code: 200
          raw: '{"status": "ok"}'
```

In this configuration:

- SPA files are served from the root path `/`
- The `/api/health` endpoint is mocked
- All other `/api/*` requests are proxied to `https://api.example.com`
