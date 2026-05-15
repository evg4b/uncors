This guide helps you migrate your UNCORS configuration files when upgrading
between versions. Breaking changes are documented with before/after examples.

## Table of Contents

 - [Version 0.5.x to 0.6.x](#version-05x-to-06x)
   
    - [TLS Certificate Configuration
      Changes](#tls-certificate-configuration-changes)
    - [Port Configuration Changes](#port-configuration-changes)
    - [Fake Response Feature Removal](#fake-response-feature-removal)
 - [Version 0.4.x to 0.5.x](#version-04x-to-05x)
 - [Older Versions](#older-versions)

---

## Version 0.5.x to 0.6.x

This is a major version with several breaking changes. Review all three sections
before upgrading.

### TLS Certificate Configuration Changes

**Breaking Change:** Global `cert-file` and `key-file` configuration properties
have been removed from the root level. TLS certificates must now use
auto-generated certificates with a local CA.

#### Why This Change?

The previous architecture required all HTTPS mappings to share the same
certificate. The new approach uses auto-generated certificates with a local CA,
providing greater flexibility, SNI support, and simpler configuration.

#### Migration Steps

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

Before using HTTPS mappings, generate a local CA once:

```bash
uncors generate-certs
```

This creates `~/.config/uncors/ca.crt`. Add this certificate to your system's
trusted certificates to avoid browser warnings.

**Key Changes:**

 1. **Remove** the global `cert-file` and `key-file` properties
 2. **Run** `uncors generate-certs` to create a local CA
 3. **Trust** the CA certificate in your system's certificate store

#### Benefits of Auto-Generated Certificates

 - Automatic certificate generation for any host
 - No need to manage certificate files
 - SNI (Server Name Indication) support - multiple hosts on the same port
 - Certificates cached in memory, regenerated only when needed

---

### Port Configuration Changes

**Breaking Change:** Global `http-port` and `https-port` configuration
properties have been removed. Ports are now specified directly in the mapping
URLs.

#### Why This Change?

The previous architecture required all HTTP mappings to share the same port and
all HTTPS mappings to share the same port. The new per-mapping port
configuration allows each mapping to listen on its own port.

#### Migration Steps

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
 2. **Add** the port number directly to the `from` URL: `protocol://host:port`
 3. Ports default to 80 for HTTP and 443 for HTTPS when omitted

#### Multiple Ports Example

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

#### Short Syntax Migration

**Old:**

```yaml
http-port: 8080
mappings:
  - http://localhost: https://github.com
```

**New:**

```yaml
mappings:
  - http://localhost:8080: https://github.com
```

#### Command-Line Arguments Migration

**Old:**

```bash
uncors --http-port 8080 --from http://localhost --to https://api.example.com
```

**New:**

```bash
uncors --from http://localhost:8080 --to https://api.example.com
```

#### Wildcard Mapping Migration

**Old:**

```yaml
http-port: 8080
mappings:
  - from: http://*.local.com
    to: https://*.example.com
```

**New:**

```yaml
mappings:
  - from: http://*.local.com:8080
    to: https://*.example.com
```

---

### Fake Response Feature Removal

**Breaking Change:** The `fake` field for generating mock responses with fake
data has been removed. Use Lua script handlers instead for dynamic response
generation.

#### Why This Change?

The Lua script handler provides a more flexible and powerful way to generate
dynamic responses. It allows for complex logic, external command execution, and
better maintainability.

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
 2. **Add** a `scripts` section with a Lua script handler
 3. **Use** the [fakedata](https://github.com/lucapette/fakedata) CLI tool or
    any other tool of your choice

#### Installing fakedata

**macOS:**

```bash
brew install lucapette/tap/fakedata
```

**Linux/macOS with Go:**

```bash
go install github.com/lucapette/fakedata@latest
```

#### Example: Array of Objects

**Old:**

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

**New:**

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

#### Advantages of the Script Handler Approach

 - **More flexible**: Execute any command-line tool, not just fake data
   generation
 - **Better control**: Use Lua logic for complex data transformations
 - **External tools**: Leverage `fakedata`, `faker-cli`, or custom scripts
 - **Full language**: Access to Lua's complete capabilities for complex
   scenarios

For more information, see the [Script Handler documentation](Script-Handler).

---

## Need Help?

If you encounter issues during migration:

 1. Check the [Configuration](Configuration) documentation for the current
    format
 2. Review the [JSON
    Schema](https://raw.githubusercontent.com/evg4b/uncors/main/schema.json) for
    configuration validation
 3. Enable debug mode: `uncors --config .uncors.yaml --debug`
 4. Report issues at [GitHub Issues](https://github.com/evg4b/uncors/issues)
