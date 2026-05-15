This guide provides practical, copy-paste ready examples for common UNCORS use
cases.

## Frontend Development with Backend API

**Scenario:** You're developing a React/Vue/Angular app locally that needs to
connect to a remote API, but CORS blocks requests.

**1. Hosts file (`/etc/hosts` or `C:\Windows\System32\drivers\etc\hosts`):**

```
127.0.0.1 app.local
```

**2. UNCORS configuration (`.uncors.yaml`):**

```yaml
mappings:
  - from: http://app.local:3000
    to: https://api.production.com
```

**3. Start UNCORS:**

```bash
uncors --config .uncors.yaml
```

**4. Configure your frontend to use the local domain:**

```javascript
// .env.local
VITE_API_URL=http://app.local:3000
// or
REACT_APP_API_URL=http://app.local:3000
```

**5. Make requests:**

```javascript
fetch("http://app.local:3000/api/users")
  .then((res) => res.json())
  .then((data) => console.log(data));
```

**Benefits:**

 - No CORS errors
 - No backend modifications needed
 - Works with any frontend framework

---

## Microservices Development

**Scenario:** You're working on a microservices architecture and want to route
different paths to different services locally.

**Hosts file:**

```
127.0.0.1 gateway.local
```

**Configuration:**

```yaml
mappings:
  - from: http://gateway.local:8000
    to: https://production-gateway.com
    rewrites:
      # Route authentication requests to auth service
      - from: /auth/{endpoint}
        to: /v1/{endpoint}
        host: auth.production.com

      # Route user requests to user service
      - from: /users/{endpoint}
        to: /api/{endpoint}
        host: users.production.com

      # Route payment requests to payment service
      - from: /payments/{endpoint}
        to: /v2/payments/{endpoint}
        host: payments.production.com
```

**Usage:**

```bash
# Authentication request → auth.production.com
curl http://gateway.local:8000/auth/login

# User request → users.production.com
curl http://gateway.local:8000/users/profile

# Payment request → payments.production.com
curl http://gateway.local:8000/payments/process
```

---

## API Mocking for Testing

**Scenario:** You need to test frontend behavior with different API responses
without affecting the real backend.

**Hosts file:**

```
127.0.0.1 api.test
```

**Configuration:**

```yaml
mappings:
  - from: http://api.test:3000
    to: https://api.production.com
    mocks:
      # Mock successful response
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

      # Mock error response
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

      # Mock slow response (network latency simulation)
      - path: /api/slow-endpoint
        method: GET
        response:
          code: 200
          delay: 3s
          headers:
            Content-Type: application/json
          raw: '{"status": "completed"}'

      # Mock paginated response
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
              "pagination": {
                "page": 1,
                "total": 10
              }
            }
```

**Test scenarios:**

```bash
# Test successful user fetch
curl http://api.test:3000/api/users/123

# Test permission error
curl -X DELETE http://api.test:3000/api/users/123

# Test slow network
curl http://api.test:3000/api/slow-endpoint

# Test pagination
curl "http://api.test:3000/api/posts?page=1"
```

---

## Local Development with Production APIs

**Scenario:** You want to use production APIs but override specific endpoints
with local data for testing.

**Hosts file:**

```
127.0.0.1 dev.local
```

**Configuration:**

```yaml
mappings:
  - from: http://dev.local:4000
    to: https://api.production.com

    # Override authentication with mock (bypass real auth)
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
              "user": {
                "id": "dev-user",
                "email": "dev@example.com"
              }
            }

    # Cache expensive endpoints
    cache:
      - /api/config/**
      - /api/metadata/**

    # Serve local static assets
    statics:
      - path: /assets
        dir: ~/projects/my-app/local-assets
```

**Usage:**

```bash
# Get mock auth token (no real authentication)
curl -X POST http://dev.local:4000/auth/token

# Use production API for data
curl http://dev.local:4000/api/users

# Serve local assets
curl http://dev.local:4000/assets/logo.png
```

---

## SPA Development with API Proxying

**Scenario:** You're building a Single-Page Application and need both local file
serving and API proxying.

**Hosts file:**

```
127.0.0.1 app.local
```

**Configuration:**

```yaml
mappings:
  - from: http://app.local:3000
    to: https://api.backend.com

    # Serve SPA files
    statics:
      - path: /
        dir: ~/projects/spa/dist
        index: index.html   # Fallback for client-side routing

    # Mock health endpoint
    mocks:
      - path: /api/health
        method: GET
        response:
          code: 200
          headers:
            Content-Type: application/json
          raw: '{"status": "ok"}'

    # Cache static API responses
    cache:
      - /api/config
      - /api/static-data/**
```

**Build and run:**

```bash
# Build SPA
npm run build  # Outputs to dist/

# Start UNCORS
uncors --config .uncors.yaml

# Access app
open http://app.local:3000
```

**Request routing:**

 - `http://app.local:3000/` → Serves `dist/index.html`
 - `http://app.local:3000/dashboard` → Serves `dist/index.html` (SPA routing)
 - `http://app.local:3000/assets/logo.png` → Serves `dist/assets/logo.png`
 - `http://app.local:3000/api/health` → Returns mock response
 - `http://app.local:3000/api/users` → Proxies to
   `https://api.backend.com/api/users`

---

## Multi-Environment Setup

**Scenario:** You need to switch between dev, staging, and production APIs
easily.

**Hosts file:**

```
127.0.0.1 api.local
```

**Configuration files:**

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

**Usage:**

```bash
uncors --config .uncors.dev.yaml      # Development
uncors --config .uncors.staging.yaml  # Staging
uncors --config .uncors.prod.yaml     # Production-like
```

**Shell aliases (optional):**

```bash
# Add to ~/.bashrc or ~/.zshrc
alias uncors-dev='uncors --config .uncors.dev.yaml'
alias uncors-staging='uncors --config .uncors.staging.yaml'
alias uncors-prod='uncors --config .uncors.prod.yaml'
```

---

## GraphQL API Development

**Scenario:** You're developing a GraphQL client and need to mock GraphQL
responses.

**Hosts file:**

```
127.0.0.1 graphql.local
```

**Configuration:**

```yaml
mappings:
  - from: http://graphql.local:4000
    to: https://api.production.com

    scripts:
      - path: /graphql
        method: POST
        script: |
          local json = require("json")

          -- Parse GraphQL request
          local body = json.decode(request.body)
          local query = body.query or ""

          -- Mock different queries
          if string.find(query, "query GetUser") then
            response.headers["Content-Type"] = "application/json"
            response:WriteHeader(200)
            response:WriteString(json.encode({
              data = {
                user = {
                  id = "123",
                  name = "Test User",
                  email = "test@example.com"
                }
              }
            }))
          elseif string.find(query, "mutation CreatePost") then
            response.headers["Content-Type"] = "application/json"
            response:WriteHeader(200)
            response:WriteString(json.encode({
              data = {
                createPost = {
                  id = "new-post-id",
                  title = "New Post",
                  createdAt = os.date("%Y-%m-%dT%H:%M:%SZ")
                }
              }
            }))
          else
            -- Forward to real API
            response:WriteHeader(502)
            response:WriteString("Query not mocked")
          end
```

**Usage:**

```bash
# Query
curl -X POST http://graphql.local:4000/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "query GetUser { user(id: \"123\") { id name email } }"}'

# Mutation
curl -X POST http://graphql.local:4000/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "mutation CreatePost { createPost(title: \"Hello\") { id title } }"}'
```

---

## WebSocket Proxying

**Scenario:** You need to proxy WebSocket connections during development.

**Hosts file:**

```
127.0.0.1 ws.local
```

**Configuration:**

```yaml
mappings:
  - from: http://ws.local:8080
    to: https://websocket.production.com
```

**Client code:**

```javascript
const ws = new WebSocket("ws://ws.local:8080/socket");

ws.onopen = () => {
  console.log("Connected");
  ws.send("Hello");
};

ws.onmessage = (event) => {
  console.log("Received:", event.data);
};
```

> [!NOTE]
> UNCORS transparently proxies WebSocket upgrade requests. No special
> configuration is needed beyond the standard HTTP mapping.

---

## Development Team Setup

**Scenario:** Your team needs a standardized UNCORS setup for consistent
development environments.

**Project structure:**

```
my-project/
├── .uncors.yaml          # Team configuration
├── .env.example          # Environment template
└── scripts/
    └── setup.sh          # Setup script
```

**`.uncors.yaml`:**

```yaml
mappings:
  - from: http://app.local:3000
    to: https://api.staging.company.com

    mocks:
      # Mock slow endpoints for better developer experience
      - path: /api/reports/generate
        method: POST
        response:
          code: 202
          delay: 100ms
          headers:
            Content-Type: application/json
          raw: '{"job_id": "mock-job-123", "status": "processing"}'

    cache:
      - /api/config/**
      - /api/constants/**
```

**`scripts/setup.sh`:**

```bash
#!/bin/bash

echo "Setting up UNCORS development environment..."

# Check if UNCORS is installed
if ! command -v uncors &> /dev/null; then
    echo "Installing UNCORS..."
    brew install evg4b/tap/uncors
fi

# Add hosts file entry
if ! grep -q "app.local" /etc/hosts; then
    echo "127.0.0.1 app.local" | sudo tee -a /etc/hosts
    echo "Added app.local to hosts file"
fi

# Start UNCORS
echo "Starting UNCORS..."
uncors --config .uncors.yaml

echo "Setup complete! Access the app at http://app.local:3000"
```

**Team onboarding:**

```bash
git clone https://github.com/company/my-project.git
cd my-project
./scripts/setup.sh
```

---

## Best Practices

 1. **Use descriptive domain names** - `app.local`, `api.local`, not
    `test1.local`
 2. **Document hosts file entries** - keep a README with required entries
 3. **Version control configuration** - commit `.uncors.yaml` to git
 4. **Environment-specific configs** - use separate files for dev/staging/prod
 5. **Mock slow endpoints** - improve developer experience with instant
    responses
 6. **Cache static data** - reduce upstream load and improve speed
 7. **Use scripts for complex logic** - keep configuration files simple

---

## Configuration Templates

### Basic Proxy

```yaml
mappings:
  - from: http://[YOUR-DOMAIN]:3000
    to: https://[TARGET-API]
```

### Proxy with Mocking

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

### SPA with API

```yaml
mappings:
  - from: http://[YOUR-DOMAIN]:3000
    to: https://[TARGET-API]
    statics:
      - path: /
        dir: [PATH-TO-BUILD]
        index: index.html
```

### Full-Featured

```yaml
debug: false
cache-config:
  expiration-time: 10m

mappings:
  - from: http://[YOUR-DOMAIN]:3000
    to: https://[TARGET-API]

    statics:
      - path: /
        dir: [BUILD-DIR]
        index: index.html

    mocks:
      - path: /api/[ENDPOINT]
        response:
          code: 200
          headers:
            Content-Type: application/json
          file: ./mocks/[FILE].json

    cache:
      - /api/**

    rewrites:
      - from: /old-api/{path}
        to: /v2/api/{path}
```

---

For more details on any of these features, see:

 - [Configuration](Configuration)
 - [Response Mocking](Response-Mocking)
 - [Static File Serving](Static-File-Serving)
 - [Script Handler](Script-Handler)
 - [Request Rewriting](Request-Rewriting)
 - [Response Caching](Response-Caching)
