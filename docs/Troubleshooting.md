# Troubleshooting

This guide helps you diagnose and resolve common issues when using UNCORS.

## Quick Diagnostics Checklist

Before diving into specific issues, verify these basics:

- [ ] UNCORS is running and showing no startup errors
- [ ] Your hosts file contains the correct domain mapping to `127.0.0.1`
- [ ] The port in your UNCORS configuration matches the port you're accessing
- [ ] Your browser/client is not using a proxy that bypasses localhost
- [ ] CORS errors are actually UNCORS-related (check browser console)

## Common Issues

### Connection Refused or Cannot Connect

**Symptoms:**

- Browser shows "Connection refused" or "Cannot connect"
- `curl` returns "Failed to connect to [domain]"

**Causes and Solutions:**

**1. UNCORS is not running**

Check if UNCORS is running:

```bash
# Check for UNCORS process
ps aux | grep uncors
```

Start UNCORS if not running:

```bash
uncors --config .uncors.yaml
```

**2. Wrong port in URL**

Verify the port matches your configuration:

```yaml
# Configuration file
mappings:
  - from: http://api.local:3000 # Port 3000
    to: https://api.example.com
```

Access URL must match:

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

Should show:

```
127.0.0.1 api.local
```

If missing, add it (see [Installation](./Installation#post-installation-hosts-file-setup)).

**4. DNS cache not flushed**

Flush DNS cache after modifying hosts file:

```bash
# macOS
sudo dscacheutil -flushcache
sudo killall -HUP mDNSResponder

# Linux (systemd)
sudo systemctl restart systemd-resolved

# Windows
ipconfig /flushdns
```

---

### HTTPS Certificate Errors

**Symptoms:**

- "NET::ERR_CERT_INVALID" in browser
- "SSL certificate problem" in curl
- "Unable to verify the first certificate"

**Causes and Solutions:**

**1. CA certificate not generated**

Generate the local CA certificate:

```bash
uncors generate-certs
```

This creates CA files in `~/.config/uncors/`:
- `ca.crt` - CA certificate
- `ca.key` - CA private key

**2. CA certificate not trusted**

Add the CA certificate to your system's trusted certificates:

**macOS:**
```bash
# Open Keychain Access
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
# Import via Certificate Manager
certutil -addstore -user "Root" %USERPROFILE%\.config\uncors\ca.crt
```

**3. Browser not using system certificates**

Some browsers maintain their own certificate stores. For Firefox:

1. Settings → Privacy & Security → Certificates → View Certificates
2. Import `~/.config/uncors/ca.crt` under "Authorities" tab

**4. CA certificate expired**

Check CA certificate validity:

```bash
openssl x509 -in ~/.config/uncors/ca.crt -noout -dates
```

If expired, regenerate:

```bash
uncors generate-certs --force
```

Then re-trust the new CA certificate.

**5. Development bypass (not recommended)**

For quick testing only:

**curl:** Use `-k` flag to ignore certificate errors:

```bash
curl -k https://api.local:8443/
```

**Browser:** Accept the certificate warning (not recommended for regular development)

---

### CORS Errors Still Appearing

**Symptoms:**

- Browser console shows CORS errors despite using UNCORS
- "Access-Control-Allow-Origin" errors

**Causes and Solutions:**

**1. Request not going through UNCORS**

Verify request is routed through UNCORS:

Check UNCORS logs (use `--debug` flag):

```bash
uncors --config .uncors.yaml --debug
```

You should see log entries for each request.

**2. OPTIONS request being forwarded instead of handled**

By default, UNCORS handles OPTIONS requests locally. If disabled, upstream server must handle them.

Check your configuration:

```yaml
mappings:
  - from: http://api.local:3000
    to: https://api.example.com
    options-handling:
      disabled: false # Should be false (default)
```

**3. Custom headers overriding CORS headers**

If you've set custom CORS headers in mocks or scripts, ensure they're correct:

```yaml
mocks:
  - path: /api/test
    response:
      code: 200
      headers:
        Access-Control-Allow-Origin: "*" # Must include this
        Access-Control-Allow-Methods: "GET, POST, OPTIONS"
      raw: "test"
```

**4. Browser cache contains old CORS responses**

Clear browser cache and reload:

- Chrome: Ctrl+Shift+Delete (Windows/Linux) or Cmd+Shift+Delete (macOS)
- Firefox: Ctrl+Shift+Delete (Windows/Linux) or Cmd+Shift+Delete (macOS)
- Or use incognito/private mode

---

### Configuration File Not Loading

**Symptoms:**

- UNCORS starts but doesn't apply configuration
- "No mappings configured" error
- Configuration changes not taking effect

**Causes and Solutions:**

**1. Wrong configuration file path**

Verify file path is correct:

```bash
ls -l .uncors.yaml  # Should exist
uncors --config .uncors.yaml --debug
```

Use absolute path if relative path fails:

```bash
uncors --config /absolute/path/to/.uncors.yaml
```

**2. YAML syntax errors**

Validate YAML syntax:

```bash
# Using Python
python -c "import yaml; yaml.safe_load(open('.uncors.yaml'))"

# Using Ruby
ruby -ryaml -e "YAML.load_file('.uncors.yaml')"
```

Common YAML mistakes:

- Incorrect indentation (use spaces, not tabs)
- Missing colons after keys
- Unquoted special characters

**3. Configuration not reloaded after changes**

UNCORS doesn't auto-reload configuration. Restart after changes:

```bash
# Stop UNCORS (Ctrl+C)
# Then restart
uncors --config .uncors.yaml
```

---

### Mocks Not Working

**Symptoms:**

- Mock responses not returned
- Requests still going to upstream server

**Causes and Solutions:**

**1. Path doesn't match**

Verify path matches exactly:

```yaml
mocks:
  - path: /api/users # Must match request path exactly
    response:
      code: 200
      raw: "mock response"
```

Test:

```bash
curl http://api.local:3000/api/users   # Matches
curl http://api.local:3000/api/users/  # Doesn't match (trailing slash)
```

Use path variables for flexibility:

```yaml
mocks:
  - path: /api/users/{id}
    response:
      code: 200
      raw: '{"id": "123"}'
```

**2. HTTP method doesn't match**

Specify method if needed:

```yaml
mocks:
  - path: /api/users
    method: POST # Only matches POST requests
    response:
      code: 201
      raw: "created"
```

**3. Mock file not found**

For file-based mocks:

```yaml
mocks:
  - path: /api/data
    response:
      code: 200
      file: ./mock-data.json # Verify this file exists
```

Check file exists:

```bash
ls -l ./mock-data.json
```

---

### Static Files Not Serving

**Symptoms:**

- 404 errors when accessing static files
- Files not loaded from local directory

**Causes and Solutions:**

**1. Directory path incorrect**

Verify directory exists:

```bash
ls -la ~/project/dist
```

Use absolute path in configuration:

```yaml
statics:
  - path: /assets
    dir: /absolute/path/to/assets # Use absolute path
```

**2. Path prefix doesn't match**

Configuration:

```yaml
statics:
  - path: /assets
    dir: ~/project/dist
```

File access must include prefix:

```bash
curl http://api.local:3000/assets/style.css  # Correct
curl http://api.local:3000/style.css         # Wrong - won't find file
```

**3. Index file not configured for SPA**

For Single-Page Applications:

```yaml
statics:
  - path: /
    dir: ~/project/build
    index: index.html # Add this for client-side routing
```

---

### High Memory or CPU Usage

**Symptoms:**

- UNCORS process consuming excessive resources
- System slowdown when UNCORS is running

**Causes and Solutions:**

**1. Cache growing too large**

Configure cache expiration:

```yaml
cache-config:
  expiration-time: 5m
  clear-time: 30m
```

Or disable caching:

```yaml
mappings:
  - from: http://api.local:3000
    to: https://api.example.com
    # Don't include 'cache:' section
```

**2. Debug logging enabled**

Disable debug mode in production:

```yaml
debug: false # Set to false
```

Or remove `--debug` flag from command line.

**3. Large response bodies being cached**

Avoid caching large responses by excluding specific paths:

```yaml
cache:
  - /api/small-responses/**
  # Don't cache /api/large-files/**
```

---

### Proxy Not Working

**Symptoms:**

- Requests fail with proxy errors
- "Proxy connection failed"

**Causes and Solutions:**

**1. Proxy configuration incorrect**

Verify proxy URL format:

```yaml
proxy: http://proxy.example.com:8080 # Correct format
```

Test proxy connectivity:

```bash
curl -x http://proxy.example.com:8080 https://google.com
```

**2. Environment variables conflicting**

UNCORS uses system proxy environment variables by default. Unset if needed:

```bash
unset HTTP_PROXY
unset HTTPS_PROXY
unset http_proxy
unset https_proxy
```

Or override in configuration:

```yaml
proxy: "" # Disable proxy
```

**3. Proxy authentication required**

If proxy requires authentication:

```yaml
proxy: http://username:password@proxy.example.com:8080
```

---

## Script Handler Issues

### Script Not Executing

**Symptoms:**

- Script handler not running
- Default response instead of script response

**Causes and Solutions:**

**1. Path or method doesn't match**

Verify path matches:

```yaml
scripts:
  - path: /api/custom
    method: GET
    script: |
      response:WriteHeader(200)
      response:WriteString("Hello")
```

**2. Script syntax error**

Test Lua syntax separately:

```bash
lua -e 'print("test")'
```

Check UNCORS logs with `--debug` for script errors.

**3. File-based script not found**

For file-based scripts:

```yaml
scripts:
  - path: /api/custom
    file: ~/scripts/handler.lua # Verify file exists
```

Check file exists:

```bash
ls -l ~/scripts/handler.lua
```

---

## Performance Issues

### Slow Response Times

**Causes and Solutions:**

**1. Enable caching for frequently accessed resources**

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

**2. Reduce upstream server latency**

Check upstream server response time:

```bash
time curl https://api.example.com/endpoint
```

**3. Use compression**

Ensure upstream server supports compression (gzip, br).

---

## Getting More Help

### Enable Debug Logging

Run with debug flag for detailed logs:

```bash
uncors --config .uncors.yaml --debug
```

### Check UNCORS Version

```bash
uncors --version
```

### Report Issues

If you've tried the above and still have issues:

1. Check [GitHub Issues](https://github.com/evg4b/uncors/issues) for similar problems
2. Create a new issue with:
   - UNCORS version
   - Operating system
   - Configuration file (sanitized)
   - Debug logs
   - Steps to reproduce

### Community Resources

- [GitHub Repository](https://github.com/evg4b/uncors)
- [Documentation](https://github.com/evg4b/uncors/wiki)
- [Issue Tracker](https://github.com/evg4b/uncors/issues)

---

## Prevention Best Practices

**1. Always use debug mode during development:**

```bash
uncors --config .uncors.yaml --debug
```

**2. Validate configuration before deploying:**

```bash
# Test with minimal configuration first
# Then gradually add features
```

**3. Keep UNCORS updated:**

```bash
# Homebrew
brew upgrade evg4b/tap/uncors

# NPM
npm update -g uncors
```

**4. Document your setup:**

- Keep notes on custom configurations
- Document hosts file entries
- Track certificate locations

**5. Use version control for configuration:**

```bash
git add .uncors.yaml
git commit -m "Add UNCORS configuration"
```
