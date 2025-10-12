# Integration Tests

This directory contains integration tests for the uncors proxy server.

## Overview

The integration test framework provides a declarative, YAML-based approach to testing uncors functionality end-to-end. Tests spin up mock backend servers, configure and start uncors, execute HTTP requests, and verify responses.

## Architecture

### Components

1. **Test Server Framework** (`framework/testserver.go`)
   - Configurable mock HTTP/HTTPS servers
   - Request recording for verification
   - Flexible endpoint configuration

2. **DNS Resolver** (`framework/dnsclient.go`)
   - Custom DNS resolution for testing
   - Support for local domain name mapping
   - HTTP client factory with DNS integration

3. **Test Case Structure** (`framework/testcase.go`)
   - YAML-based test case definitions
   - Declarative test configuration
   - Test case loader

4. **Test Runner** (`framework/runner.go`)
   - Test orchestration and lifecycle management
   - Uncors configuration generation
   - Backend server initialization
   - Request execution and response verification

## Test Case Format

Test cases are defined in YAML files in the `testcases/` directory:

```yaml
name: "Test Name"
description: "Test description"

backend:
  port: 9001  # Optional: specific port
  tls: false  # Optional: enable TLS
  endpoints:
    - path: /api/endpoint
      method: GET
      response:
        status: 200
        headers:
          Content-Type: application/json
        body: '{"key":"value"}'

uncors:
  http-port: 3000
  mappings:
    - from: http://localhost:3000
      to: "{{backend}}"  # {{backend}} is replaced with actual backend URL
      mocks:  # Optional: mock specific endpoints
        - path: /api/mock
          response:
            code: 200
            raw: '{"mocked":true}'

tests:
  - name: "Test case 1"
    request:
      method: GET
      path: /api/endpoint
      headers:  # Optional
        X-Custom-Header: value
      body: ''  # Optional
    expected:
      status: 200
      body: '{"key":"value"}'  # Optional: exact match
      body-contains:  # Optional: substring matches
        - "key"
      body-json:  # Optional: JSON field matches
        key: value
      headers:  # Optional: exact header matches
        Content-Type: application/json
      headers-exist:  # Optional: header existence checks
        - Content-Type
    dns:  # Optional: custom DNS mappings
      example.local: 127.0.0.1
```

## Running Tests

### Run all integration tests
```bash
make test-integration
```

### Run all tests (unit + integration)
```bash
make test-all
```

### Run integration tests directly with go
```bash
go test -v ./tests/integration/...
```

### Run a specific test case
```bash
go test -v ./tests/integration/... -run "TestIntegration/Test_Name"
```

### Skip integration tests (run unit tests only)
```bash
make test
# or
go test -short ./...
```

## Writing New Tests

1. Create a new YAML file in `testcases/` directory
2. Follow the test case format above
3. Name the file descriptively (e.g., `13-my-feature.yaml`)
4. Run tests to verify

### Tips

- Use `{{backend}}` placeholder in mappings to reference the mock backend server URL
- Tests run in parallel by test case name (separate uncors instances)
- Each test case gets its own HTTP port to avoid conflicts
- Backend servers are automatically started and stopped
- Uncors configuration is generated automatically

## Test Coverage

Current test cases cover:

1. **Basic Proxy** - Simple HTTP proxying
2. **CORS Headers** - CORS header replacement
3. **HTTP Methods** - GET, POST, PUT, DELETE, PATCH
4. **Custom Headers** - Header forwarding
5. **Status Codes** - 2xx, 3xx, 4xx, 5xx responses
6. **Response Bodies** - JSON, text, HTML, empty, large
7. **Mocking** - Uncors mock responses
8. **Query Parameters** - Parameter forwarding
9. **Request Bodies** - JSON, empty, large bodies
10. **Content Types** - JSON, XML, form-encoded
11. **OPTIONS Preflight** - CORS preflight requests
12. **Error Handling** - Backend errors, timeouts, 404s

## Troubleshooting

### Tests hang or timeout
- Check if port is already in use
- Verify uncors binary builds successfully
- Check uncors logs in test output

### DNS resolution issues
- Verify DNS mappings in test case
- Check that localhost resolves to 127.0.0.1

### Backend connection failures
- Ensure backend endpoints are correctly configured
- Verify backend server starts successfully
- Check for port conflicts

## Future Enhancements

- Support for HTTPS/TLS testing
- Static file serving tests
- Cache configuration tests
- Rewrite rule tests
- Wildcard mapping tests
- Multi-mapping scenarios
- Performance benchmarks
