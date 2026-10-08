Complete configurations for common setups. Each one lists the hosts file
entries it needs; see [Hosts file setup](Installation#hosts-file-setup).

## Frontend against a remote API

You develop a frontend locally and the remote API blocks it with CORS.

Hosts file:

```
127.0.0.1 api.local
```

`.uncors.yaml`:

```yaml
mappings:
  - from: http://api.local:3000
    to: https://api.production.com
```

Start UNCORS:

```bash
uncors --config .uncors.yaml
```

Point the frontend at the local address, for example in `.env.local`:

```bash
VITE_API_URL=http://api.local:3000
# or
REACT_APP_API_URL=http://api.local:3000
```

Requests now go through UNCORS, which adds CORS headers to every response:

```javascript
fetch("http://api.local:3000/api/users")
  .then((res) => res.json())
  .then((data) => console.log(data));
```

## Mocking API responses for tests

You want to test how the frontend handles specific responses without changing
the real backend.

Hosts file:

```
127.0.0.1 api.test
```

Configuration:

```yaml
mappings:
  - from: http://api.test:3000
    to: https://api.production.com
    mocks:
      # Successful response
      - path: /api/users/{id}
        method: GET
        response:
          code: 200
          headers:
            Content-Type: application/json
          raw: |
            {
              "id": "123",
              "name": "Test User",
              "email": "test@example.com",
              "role": "admin"
            }

      # Error response
      - path: /api/users/{id}
        method: DELETE
        response:
          code: 403
          headers:
            Content-Type: application/json
          raw: |
            {
              "error": "Permission denied",
              "code": "INSUFFICIENT_PERMISSIONS"
            }

      # Slow response
      - path: /api/slow-endpoint
        method: GET
        response:
          code: 200
          delay: 3s
          headers:
            Content-Type: application/json
          raw: '{"status": "completed"}'

      # First page of a list; other pages go to the real API
      - path: /api/posts
        method: GET
        queries:
          page: "1"
        response:
          code: 200
          headers:
            Content-Type: application/json
          raw: |
            {
              "data": [
                {"id": 1, "title": "Post 1"},
                {"id": 2, "title": "Post 2"}
              ],
              "pagination": {"page": 1, "total": 10}
            }
```

Try it:

```bash
curl http://api.test:3000/api/users/123
curl -X DELETE http://api.test:3000/api/users/123
curl http://api.test:3000/api/slow-endpoint
curl "http://api.test:3000/api/posts?page=1"
```

## Production API with local overrides

You use the production API but replace a few endpoints and assets with local
versions.

Hosts file:

```
127.0.0.1 dev.local
```

Configuration:

```yaml
mappings:
  - from: http://dev.local:4000
    to: https://api.production.com

    # Skip the real login
    mocks:
      - path: /auth/token
        method: POST
        response:
          code: 200
          headers:
            Content-Type: application/json
          raw: |
            {
              "token": "dev-token-12345",
              "expires_in": 3600,
              "user": {"id": "dev-user", "email": "dev@example.com"}
            }

    # Cache endpoints that are slow and rarely change
    cache:
      - /api/config/**
      - /api/metadata/**

    # Local copies of some assets; missing files come from production
    statics:
      - path: /assets
        dir: ./local-assets
```

```bash
curl -X POST http://dev.local:4000/auth/token # mock
curl http://dev.local:4000/api/users          # production API
curl http://dev.local:4000/assets/logo.png    # ./local-assets/logo.png if it exists
```

## Single-page app with an API

You serve a local build of a single-page app and its API comes from a remote
server. A static directory with `index` at `/` answers every request on its
host, so the app and the API need separate host names.

Hosts file:

```
127.0.0.1 app.local
127.0.0.1 api.local
```

Configuration:

```yaml
mappings:
  # The app
  - from: http://app.local:3000
    to: https://www.example.com
    statics:
      - path: /
        dir: ./dist
        index: index.html

  # The API
  - from: http://api.local:3000
    to: https://api.backend.com
    mocks:
      - path: /api/health
        method: GET
        response:
          code: 200
          headers:
            Content-Type: application/json
          raw: '{"status": "ok"}'
    cache:
      - /api/config
      - /api/static-data/**
```

Build the app with its API URL set to `http://api.local:3000`, then start
UNCORS:

```bash
npm run build # writes ./dist
uncors --config .uncors.yaml
```

| Request                                  | Result                                           |
| ---------------------------------------- | ------------------------------------------------ |
| `http://app.local:3000/`                 | `./dist/index.html`                              |
| `http://app.local:3000/dashboard`        | `./dist/index.html` (client-side route)          |
| `http://app.local:3000/assets/logo.png`  | `./dist/assets/logo.png`                         |
| `http://api.local:3000/api/health`       | The mock                                         |
| `http://api.local:3000/api/users`        | Proxied to `https://api.backend.com/api/users`   |

## Several backends behind one local host

You want one local host name to send different path prefixes to different
services.

Hosts file:

```
127.0.0.1 gateway.local
```

Configuration:

```yaml
mappings:
  - from: https://gateway.local:8443
    to: https://production-gateway.com
    rewrites:
      - from: /auth/{endpoint}
        to: /v1/{endpoint}
        host: auth.production.com
      - from: /users/{endpoint}
        to: /api/{endpoint}
        host: users.production.com
      - from: /payments/{endpoint}
        to: /v2/payments/{endpoint}
        host: payments.production.com
```

The source is HTTPS because a rewritten host is called with the scheme of the
incoming request. Run `uncors generate-certs` once and trust the CA first; see
[HTTPS configuration](Configuration#https-configuration).

```bash
curl https://gateway.local:8443/auth/login       # https://auth.production.com/v1/login
curl https://gateway.local:8443/users/profile    # https://users.production.com/api/profile
curl https://gateway.local:8443/payments/process # https://payments.production.com/v2/payments/process
curl https://gateway.local:8443/status           # https://production-gateway.com/status
```

Each `{endpoint}` captures one path segment, and the query string is not
forwarded. See [Request Rewriting](Request-Rewriting).

If the services don't need to share a host name, separate mappings are
simpler and keep the full path and query string:

```yaml
mappings:
  - http://auth.local:8000: https://auth.production.com
  - http://users.local:8000: https://users.production.com
  - http://payments.local:8000: https://payments.production.com
```

## One configuration per environment

You switch between development, staging, and production APIs.

Hosts file:

```
127.0.0.1 api.local
```

```yaml
# .uncors.dev.yaml
mappings:
  - from: http://api.local:3000
    to: https://api.dev.example.com
    mocks:
      - path: /debug/info
        response:
          code: 200
          raw: '{"env": "development"}'
```

```yaml
# .uncors.staging.yaml
mappings:
  - from: http://api.local:3000
    to: https://api.staging.example.com
    cache:
      - /api/**
```

```yaml
# .uncors.prod.yaml
mappings:
  - from: http://api.local:3000
    to: https://api.example.com
    cache:
      - /api/config/**
      - /api/metadata/**
```

```bash
uncors --config .uncors.dev.yaml
uncors --config .uncors.staging.yaml
uncors --config .uncors.prod.yaml
```

Shell aliases save some typing:

```bash
alias uncors-dev='uncors --config .uncors.dev.yaml'
alias uncors-staging='uncors --config .uncors.staging.yaml'
alias uncors-prod='uncors --config .uncors.prod.yaml'
```

## Mocking a GraphQL API

All GraphQL requests go to one path, so a mock can't tell queries apart. A
script can look at the query instead.

Hosts file:

```
127.0.0.1 graphql.local
```

Configuration:

```yaml
mappings:
  - from: http://graphql.local:4000
    to: https://api.production.com
    scripts:
      - path: /graphql
        method: POST
        script: |
          local json = require("json")
          response.headers["Content-Type"] = "application/json"

          local body = json.decode(request.body)
          local query = (type(body) == "table" and body.query) or ""

          if string.find(query, "query GetUser", 1, true) then
            response:WriteHeader(200)
            response:WriteString(json.encode({
              data = {
                user = {id = "123", name = "Test User", email = "test@example.com"}
              }
            }))
          elseif string.find(query, "mutation CreatePost", 1, true) then
            response:WriteHeader(200)
            response:WriteString(json.encode({
              data = {
                createPost = {
                  id = "new-post-id",
                  title = "New Post",
                  createdAt = os.date("!%Y-%m-%dT%H:%M:%SZ")
                }
              }
            }))
          else
            response:WriteHeader(400)
            response:WriteString(json.encode({
              errors = {{message = "This query is not mocked"}}
            }))
          end
```

The script handles every `POST /graphql` request, so queries it doesn't
recognize are not forwarded to the real API.

```bash
curl -X POST http://graphql.local:4000/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "query GetUser { user(id: \"123\") { id name email } }"}'

curl -X POST http://graphql.local:4000/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "mutation CreatePost { createPost(title: \"Hello\") { id title } }"}'
```

## Shared team setup

You want everyone on the team to run the same setup.

```
my-project/
├── .uncors.yaml
└── scripts/
    └── setup.sh
```

`.uncors.yaml`:

```yaml
mappings:
  - from: http://app.local:3000
    to: https://api.staging.company.com
    mocks:
      # Answer a slow report endpoint right away
      - path: /api/reports/generate
        method: POST
        response:
          code: 202
          headers:
            Content-Type: application/json
          raw: '{"job_id": "mock-job-123", "status": "processing"}'
    cache:
      - /api/config/**
      - /api/constants/**
```

`scripts/setup.sh`:

```bash
#!/bin/bash
set -e

if ! command -v uncors > /dev/null; then
  echo "Installing UNCORS..."
  brew install evg4b/tap/uncors
fi

if ! grep -q "app.local" /etc/hosts; then
  echo "127.0.0.1 app.local" | sudo tee -a /etc/hosts
fi

echo "Starting UNCORS. The app is available at http://app.local:3000"
exec uncors --config .uncors.yaml
```

New team members run:

```bash
git clone https://github.com/company/my-project.git
cd my-project
./scripts/setup.sh
```

## Not supported: WebSocket

UNCORS does not proxy WebSocket connections. Connect to WebSocket endpoints
directly; browsers don't apply CORS to WebSocket connections.

## Templates

Basic proxy:

```yaml
mappings:
  - from: http://[YOUR-DOMAIN]:3000
    to: https://[TARGET-API]
```

Proxy with a mock:

```yaml
mappings:
  - from: http://[YOUR-DOMAIN]:3000
    to: https://[TARGET-API]
    mocks:
      - path: /api/[ENDPOINT]
        response:
          code: 200
          headers:
            Content-Type: application/json
          raw: "[JSON-RESPONSE]"
```

Single-page app and API:

```yaml
mappings:
  - from: http://[APP-DOMAIN]:3000
    to: https://[APP-ORIGIN]
    statics:
      - path: /
        dir: [BUILD-DIR]
        index: index.html
  - from: http://[API-DOMAIN]:3000
    to: https://[TARGET-API]
```

Most features in one file:

```yaml
cache-config:
  expiration-time: 10m

mappings:
  - from: http://[APP-DOMAIN]:3000
    to: https://[APP-ORIGIN]
    statics:
      - path: /
        dir: [BUILD-DIR]
        index: index.html

  - from: http://[API-DOMAIN]:3000
    to: https://[TARGET-API]
    mocks:
      - path: /api/[ENDPOINT]
        response:
          code: 200
          headers:
            Content-Type: application/json
          raw: "[JSON-RESPONSE]"
    cache:
      - /api/**
    rewrites:
      - from: /old-api/{resource}
        to: /v2/api/{resource}
    har: ./recordings/api.har
```

More details on each feature:

- [Configuration](Configuration)
- [Response Mocking](Response-Mocking)
- [Static File Serving](Static-File-Serving)
- [Script Handler](Script-Handler)
- [Request Rewriting](Request-Rewriting)
- [Response Caching](Response-Caching)
- [HAR Collector](HAR-Collector)
