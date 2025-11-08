# Migration Guide

This guide helps you migrate your UNCORS configuration files to the latest version. Breaking changes are documented here with examples showing how to update your configuration.

## Version 0.5.x to 0.6.x

### TLS Certificate Configuration Changes

**Breaking Change:** Global `cert-file` and `key-file` configuration properties have been removed from the root level. TLS certificates must now use auto-generated certificates with a local CA.

#### Why This Change?

The previous architecture required all HTTPS mappings to share the same certificate. The new approach uses auto-generated certificates with a local CA, providing greater flexibility and simpler configuration.

#### TLS Configuration Migration Steps

**Old Configuration (v0.5.x and earlier):**

```yaml
cert-file: ~/certs/server.crt
key-file: ~/certs/server.key
mappings:
  - from: https://app.local:8443
    to: https://api.example.com
  - from: https://admin.local:8443
    to: https://admin.example.com
```

**New Configuration (v0.6.x):**

```yaml
mappings:
  - from: https://app.local:8443
    to: https://api.example.com
  - from: https://admin.local:8443
    to: https://admin.example.com
```

Before using HTTPS mappings, you need to create a local CA:

```bash
uncors generate-certs
```

This will create a CA certificate in `~/.config/uncors/ca.crt`. Add this certificate to your system's trusted certificates to avoid browser warnings.

**Key Changes:**

1. **Remove** the global `cert-file` and `key-file` properties from root level
2. **Generate** a local CA using `uncors generate-certs`
3. **Trust** the CA certificate in your system's certificate store

#### Benefits of Auto-Generated Certificates

- Automatic certificate generation for any host
- No need to manage certificate files
- Better security through automatic certificate management
- Support for SNI (Server Name Indication) for multiple hosts on the same port

### Port Configuration Changes

**Breaking Change:** Global `http-port` and `https-port` configuration properties have been removed. Ports are now specified directly in the mapping URLs.

#### Why This Change?

The previous architecture required all HTTP mappings to share the same port and all HTTPS mappings to share the same port. The new per-mapping port configuration allows each mapping to listen on its own port, providing greater flexibility for complex development setups.

#### Port Configuration Migration Steps

**Old Configuration (v0.5.x and earlier):**

```yaml
http-port: 8080
https-port: 8443
mappings:
  - from: http://localhost
    to: https://api.example.com
  - from: https://secure-app
    to: https://backend.example.com
```

**New Configuration (v0.6.x):**

```yaml
mappings:
  - from: http://localhost:8080
    to: https://api.example.com
  - from: https://secure-app:8443
    to: https://backend.example.com
```

**Key Changes:**

1. **Remove** the global `http-port` and `https-port` properties
2. **Add** the port number directly to the `from` URL using the format `protocol://host:port`
3. If no port is specified, defaults are used (80 for HTTP, 443 for HTTPS)

#### Multiple Ports Support

The new architecture enables each mapping to use a different port:

```yaml
mappings:
  - from: http://api.local:3000
    to: https://api.example.com
  - from: http://admin.local:4000
    to: https://admin.example.com
  - from: https://secure.local:8443
    to: https://backend.example.com
```

This configuration starts three separate servers:

- HTTP server on port 3000 for `api.local`
- HTTP server on port 4000 for `admin.local`
- HTTPS server on port 8443 for `secure.local`

#### Using Default Ports

If you want to use the default ports (80 for HTTP, 443 for HTTPS), you can omit the port number:

```yaml
mappings:
  - from: http://localhost
    to: https://api.example.com
  - from: https://secure-app
    to: https://backend.example.com
```

#### Short Syntax Migration

**Old Configuration:**

```yaml
http-port: 8080
mappings:
  - http://localhost: https://github.com
```

**New Configuration:**

```yaml
mappings:
  - http://localhost:8080: https://github.com
```

#### Command-Line Arguments Migration

**Old CLI Usage:**

```bash
uncors --http-port 8080 --from http://localhost --to https://api.example.com
```

**New CLI Usage:**

```bash
uncors --from http://localhost:8080 --to https://api.example.com
```

#### Wildcard Mapping Migration

**Old Configuration:**

```yaml
http-port: 8080
mappings:
  - from: http://*.local.com
    to: https://*.example.com
```

**New Configuration:**

```yaml
mappings:
  - from: http://*.local.com:8080
    to: https://*.example.com
```

#### Complex Configuration Example

Here's a complete example showing migration of a complex configuration:

**Before:**

```yaml
debug: true
http-port: 8080
https-port: 8443
proxy: http://proxy.example.com:3128
cert-file: ~/certs/server.crt
key-file: ~/certs/server.key

mappings:
  - from: http://api.local
    to: https://api.example.com
    mocks:
      - path: /test
        response:
          code: 200
          raw: "Test response"
  - from: https://secure.local
    to: https://secure.example.com
    statics:
      - path: /static
        dir: ./public
```

**After (v0.6.x):**

```yaml
debug: true
proxy: http://proxy.example.com:3128

mappings:
  - from: http://api.local:8080
    to: https://api.example.com
    mocks:
      - path: /test
        response:
          code: 200
          raw: "Test response"
  - from: https://secure.local:8443
    to: https://secure.example.com
    statics:
      - path: /static
        dir: ./public
```

> [!NOTE]
> In v0.7.x and later, the `cert-file` and `key-file` fields shown in the "Before" example are no longer supported.

#### Schema Validation

If you're using JSON Schema validation in your IDE, the schema will automatically detect and flag the old `http-port` and `https-port` properties as errors, helping you identify configurations that need to be migrated.

**Error Example:**

```yaml
http-port: 8080 # Error: Additional property http-port is not allowed
mappings:
  - from: http://localhost
    to: https://github.com
```

#### Troubleshooting

**Problem:** UNCORS fails to start with "Additional property" error

**Solution:** Remove `http-port` and `https-port` from your configuration file and add ports directly to the `from` URLs in your mappings.

---

**Problem:** My application can't connect to the old port

**Solution:** Update the port in your application's configuration to match the port specified in the `from` URL. For example, if you changed from `http-port: 8080` with `from: http://localhost` to `from: http://localhost:3000`, update your application to connect to `http://localhost:3000`.

---

**Problem:** I need multiple mappings on the same port

**Solution:** You can have multiple mappings on the same port by specifying the same port number in different `from` URLs:

```yaml
mappings:
  - from: http://api.local:8080
    to: https://api.example.com
  - from: http://admin.local:8080
    to: https://admin.example.com
```

Both `api.local` and `admin.local` will be served on port 8080.

### Fake Response Feature Removal

**Breaking Change:** The `fake` field for generating mock responses with fake data has been removed. Use Lua script handlers instead for dynamic response generation.

#### Why This Change?

The Lua script handler provides a more flexible and powerful way to generate dynamic responses, including fake data. It allows for complex logic, external command execution, and better maintainability.

#### Migration Steps

**Old Configuration (v0.5.x and earlier):**

```yaml
mappings:
  - from: http://localhost:8080
    to: https://api.example.com
    mocks:
      - path: /api/users
        response:
          code: 200
          fake:
            type: object
            properties:
              login:
                type: email
              username:
                type: username
          seed: 12345
```

**New Configuration (v0.6.x) - Using Script Handler:**

```yaml
mappings:
  - from: http://localhost:8080
    to: https://api.example.com
    scripts:
      - path: /api/users
        script: |
          local handle = io.popen("fakedata --format=ndjson --limit 1 login=email username=username")
          local output = handle:read("*a")
          handle:close()

          response.headers["Content-Type"] = "application/json"
          response:WriteHeader(200)
          response:WriteString(output)
```

**Key Changes:**

1. **Remove** the `mocks` section with `fake` and `seed` properties
2. **Add** a `scripts` section with Lua script handler
3. **Use** the [fakedata](https://github.com/lucapette/fakedata) command-line tool to generate fake data or any other CLI tool of your choice

#### Installing fakedata CLI Tool

To use the fakedata tool for generating fake data:

**macOS:**

```bash
brew install lucapette/tap/fakedata
```

**Linux/macOS with Go:**

```bash
go install github.com/lucapette/fakedata@latest
```

**Usage Example:**

```bash
fakedata --format=ndjson --limit 1 login=email username=username
```

#### Example: Array of Objects

**Old Configuration:**

```yaml
response:
  code: 200
  fake:
    type: array
    item:
      type: object
      properties:
        name:
          type: name
        email:
          type: email
    count: 5
```

**New Configuration:**

```yaml
scripts:
  - path: /api/users
    script: |
      local handle = io.popen("fakedata --format=ndjson --limit 5 name=name email=email")
      local output = handle:read("*a")
      handle:close()

      -- Convert NDJSON to JSON array
      local lines = {}
      for line in output:gmatch("[^\n]+") do
          table.insert(lines, line)
      end
      local result = "[" .. table.concat(lines, ",") .. "]"

      response.headers["Content-Type"] = "application/json"
      response:WriteHeader(200)
      response:WriteString(result)
```

#### Advantages of Script Handler Approach

- **More Flexible**: Execute any command-line tool, not just fake data generation
- **Better Control**: Use Lua logic for complex data transformations
- **Reproducible**: Use environment variables or files to control seed values
- **External Tools**: Leverage existing CLI tools like `fakedata`, `faker-cli`, or custom scripts
- **Full Programming Language**: Access to Lua's full capabilities for complex scenarios

For more information about the Script Handler feature, see the [Script Handler documentation](./Script-Handler).

## Need Help?

If you encounter issues during migration or have questions:

1. Check the [Configuration](./Configuration) documentation for detailed information about the new configuration format
2. Review the [JSON Schema](https://raw.githubusercontent.com/evg4b/uncors/main/schema.json) for configuration validation
3. Report issues at [GitHub Issues](https://github.com/evg4b/uncors/issues)
