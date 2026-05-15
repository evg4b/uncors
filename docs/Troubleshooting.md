This guide helps you diagnose and resolve common issues when using UNCORS.

## Quick Diagnostics Checklist

Before diving into specific issues, verify these basics:

- [ ] UNCORS is running and showing no startup errors
- [ ] Your hosts file contains the correct domain mapping to `127.0.0.1`
- [ ] The port in your UNCORS configuration matches the port you're accessing
- [ ] Your browser/client is not using a proxy that bypasses localhost
- [ ] CORS errors are actually UNCORS-related (check browser console)

---

## Connection Refused or Cannot Connect

**Symptoms:**

- Browser shows "Connection refused" or "Cannot connect"
- `curl` returns "Failed to connect to [domain]"

**1. UNCORS is not running**

```bash
# Check for UNCORS process
ps aux | grep uncors

# Start UNCORS if not running
uncors --config .uncors.yaml
```

**2. Wrong port in URL**

Verify the port matches your configuration:

```yaml
mappings:
  - from: http://api.local:3000  # Port 3000
    to: https://api.example.com
```

```bash
curl http://api.local:3000/  # Correct
curl http://api.local:8080/  # Wrong - will fail
```

**3. Hosts file not configured**

Verify hosts file entry:

```bash
# macOS/Linux
cat /etc/hosts | grep api.local

# Windows (PowerShell)
Get-Content C:\Windows\System32\drivers\etc\hosts | Select-String api.local
```

Expected output: `127.0.0.1 api.local`. If missing, see [Installation → Hosts File Setup](Installation#post-installation-hosts-file-setup).

**4. DNS cache not flushed**

After modifying the hosts file, flush the DNS cache:

```bash
# macOS
sudo dscacheutil -flushcache && sudo killall -HUP mDNSResponder

# Linux (systemd)
sudo systemctl restart systemd-resolved
```

```cmd
# Windows
ipconfig /flushdns
```

---

## HTTPS Certificate Errors

**Symptoms:**

- "NET::ERR_CERT_INVALID" in browser
- "SSL certificate problem" in curl
- "Unable to verify the first certificate"

**1. CA certificate not generated**

```bash
uncors generate-certs
```

This creates `~/.config/uncors/ca.crt` and `~/.config/uncors/ca.key`.

**2. CA certificate not trusted**

**macOS:**

```bash
open ~/.config/uncors/ca.crt
# Set to "Always Trust" in Keychain Access
```

**Linux:**

```bash
sudo cp ~/.config/uncors/ca.crt /usr/local/share/ca-certificates/uncors-ca.crt
sudo update-ca-certificates
```

**Windows:**

```powershell
certutil -addstore -user "Root" %USERPROFILE%\.config\uncors\ca.crt
```

**3. Browser not using system certificates**

Firefox maintains its own certificate store:

1. Settings → Privacy & Security → Certificates → View Certificates
2. Import `~/.config/uncors/ca.crt` under the "Authorities" tab

**4. CA certificate expired**

```bash
# Check expiry
openssl x509 -in ~/.config/uncors/ca.crt -noout -dates

# Regenerate if expired
uncors generate-certs --force
```

Then re-trust the new certificate.

**5. Development bypass (not recommended for regular use)**

```bash
# curl: ignore certificate errors
curl -k https://api.local:8443/
```

---

## CORS Errors Still Appearing

**Symptoms:**

- Browser console shows CORS errors despite using UNCORS
- "Access-Control-Allow-Origin" header errors

**1. Request not going through UNCORS**

Enable debug logging and verify requests appear in the output:

```bash
uncors --config .uncors.yaml --debug
```

**2. OPTIONS request being forwarded instead of handled**

By default, UNCORS handles OPTIONS requests locally. If disabled, the upstream server must handle them:

```yaml
mappings:
  - from: http://api.local:3000
    to: https://api.example.com
    options-handling:
      disabled: false  # Must be false (default) for UNCORS to handle preflight
```

**3. Custom headers overriding CORS headers**

If you've set custom CORS headers in mocks or scripts, verify they're correct:

```yaml
mocks:
  - path: /api/test
    response:
      code: 200
      headers:
        Access-Control-Allow-Origin: "*"
        Access-Control-Allow-Methods: "GET, POST, OPTIONS"
      raw: "test"
```

**4. Browser cache contains old CORS responses**

Clear browser cache (Chrome: `Ctrl+Shift+Delete`, macOS: `Cmd+Shift+Delete`) or use an incognito/private window.

---

## Configuration File Not Loading

**Symptoms:**

- UNCORS starts but doesn't apply configuration
- "No mappings configured" error
- Configuration changes not taking effect

**1. Wrong configuration file path**

```bash
ls -l .uncors.yaml  # Should exist
uncors --config .uncors.yaml --debug
```

Use an absolute path if a relative path fails:

```bash
uncors --config /absolute/path/to/.uncors.yaml
```

**2. YAML syntax errors**

Validate YAML syntax:

```bash
python3 -c "import yaml; yaml.safe_load(open('.uncors.yaml'))"
```

Common YAML mistakes:

- Incorrect indentation (use spaces, not tabs)
- Missing colons after keys
- Unquoted special characters

**3. Configuration not reloaded after changes**

UNCORS does not auto-reload configuration. Restart after changes:

```bash
# Stop UNCORS with Ctrl+C, then restart
uncors --config .uncors.yaml
```

---

## Mocks Not Working

**Symptoms:**

- Mock responses not returned
- Requests still going to the upstream server

**1. Path doesn't match exactly**

```yaml
mocks:
  - path: /api/users   # Matches /api/users but NOT /api/users/
    response:
      code: 200
      raw: "mock response"
```

Use path variables for flexibility:

```yaml
mocks:
  - path: /api/users/{id}
    response:
      code: 200
      raw: '{"id": "123"}'
```

**2. HTTP method filter too restrictive**

If you specify a method, only that method is matched:

```yaml
mocks:
  - path: /api/users
    method: POST   # Only matches POST requests; GET requests pass through
```

**3. Mock file not found**

```yaml
mocks:
  - path: /api/data
    response:
      code: 200
      file: ./mock-data.json   # Verify this file exists
```

```bash
ls -l ./mock-data.json
```

---

## Static Files Not Serving

**Symptoms:**

- 404 errors when accessing static files
- Files not loaded from local directory

**1. Directory path incorrect**

```bash
ls -la ~/project/dist
```

Use an absolute path in the configuration if needed:

```yaml
statics:
  - path: /assets
    dir: /absolute/path/to/assets
```

**2. Path prefix doesn't match**

With the configuration below, the URL must include the `/assets` prefix:

```yaml
statics:
  - path: /assets
    dir: ~/project/dist
```

```bash
curl http://api.local:3000/assets/style.css   # Correct
curl http://api.local:3000/style.css          # Wrong - prefix missing
```

**3. Missing index file for SPA routing**

```yaml
statics:
  - path: /
    dir: ~/project/build
    index: index.html   # Required for client-side routing
```

---

## High Memory or CPU Usage

**Symptoms:**

- UNCORS process consuming excessive resources
- System slowdown when UNCORS is running

**1. Cache growing too large**

Configure a shorter expiration time or smaller max size:

```yaml
cache-config:
  expiration-time: 5m
  max-size: 52428800   # 50 MB
```

Or disable caching for this mapping by omitting the `cache:` section entirely.

**2. Debug logging enabled**

```yaml
debug: false
```

**3. Large response bodies being cached**

Only cache paths that return small responses:

```yaml
cache:
  - /api/small-responses/**
  # Avoid caching /api/large-files/**
```

---

## Proxy Not Working

**Symptoms:**

- Requests fail with proxy errors
- "Proxy connection failed"

**1. Proxy URL format incorrect**

```yaml
proxy: http://proxy.example.com:8080   # Correct format
```

Test connectivity:

```bash
curl -x http://proxy.example.com:8080 https://google.com
```

**2. Environment variables conflicting**

UNCORS reads system proxy environment variables by default. Unset them if needed:

```bash
unset HTTP_PROXY HTTPS_PROXY http_proxy https_proxy
```

Or override in the configuration:

```yaml
proxy: ""   # Disable proxy
```

**3. Proxy requires authentication**

```yaml
proxy: http://username:password@proxy.example.com:8080
```

---

## Script Handler Issues

**Script Not Executing**

**Symptoms:** Script handler not running; default response returned instead.

**1. Path or method filter doesn't match**

```yaml
scripts:
  - path: /api/custom
    method: GET
    script: |
      response:WriteHeader(200)
      response:WriteString("Hello")
```

Verify the path and method match your request exactly.

**2. Script syntax error**

Enable debug logging to see script errors:

```bash
uncors --config .uncors.yaml --debug
```

**3. File-based script not found**

```yaml
scripts:
  - path: /api/custom
    file: ~/scripts/handler.lua   # Verify file exists
```

```bash
ls -l ~/scripts/handler.lua
```

---

## Performance Issues

**Slow response times:**

1. Enable caching for frequently accessed resources

```yaml
cache-config:
  expiration-time: 10m
  methods: [GET]

mappings:
  - from: http://api.local:3000
    to: https://api.example.com
    cache:
      - /api/**
```

2. Check upstream server response time directly

```bash
time curl https://api.example.com/endpoint
```

3. Ensure the upstream server supports compression (gzip, br)

---

## Getting More Help

### Enable Debug Logging

```bash
uncors --config .uncors.yaml --debug
```

### Check UNCORS Version

```bash
uncors --version
```

### Report Issues

If you've tried the above and still have problems, create an issue at [GitHub Issues](https://github.com/evg4b/uncors/issues) with:

- UNCORS version (`uncors --version`)
- Operating system
- Configuration file (with sensitive values removed)
- Debug logs
- Steps to reproduce

### Community Resources

- [GitHub Repository](https://github.com/evg4b/uncors)
- [Issue Tracker](https://github.com/evg4b/uncors/issues)

---

## Prevention Best Practices

1. **Always use debug mode during initial setup** to see what requests are being handled
2. **Validate your YAML** before starting - syntax errors produce confusing startup behavior
3. **Keep UNCORS updated:**

```bash
brew upgrade evg4b/tap/uncors   # Homebrew
npm update -g uncors            # NPM
```

4. **Document your setup** - note hosts file entries, certificate locations, and config paths
5. **Version control your configuration** - commit `.uncors.yaml` alongside your project
