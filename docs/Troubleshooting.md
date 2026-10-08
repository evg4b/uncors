Common problems with UNCORS and how to fix them.

## First checks

- UNCORS is running and printed no errors at startup.
- The hosts file maps your local domain to `127.0.0.1`.
- The URL you open uses the same host and port as the mapping's `from`.
- Your browser or HTTP client isn't sending local traffic through a proxy.
- The failing request shows up in the UNCORS console. If it doesn't, it never
  reached UNCORS.

## Connection refused

The browser shows "Connection refused", or `curl` says "Failed to connect".

UNCORS may not be running. Start it and watch for errors:

```bash
uncors --config .uncors.yaml
```

The port may be wrong. UNCORS listens on the port from the `from` URL, or on 80
or 443 when `from` has no port:

```yaml
mappings:
  - from: http://api.local:3000 # listens on 3000
    to: https://api.example.com
```

```bash
curl http://api.local:3000/ # works
curl http://api.local:8080/ # connection refused
```

The hosts file entry may be missing:

```bash
# macOS/Linux
grep api.local /etc/hosts
```

```powershell
# Windows
Get-Content C:\Windows\System32\drivers\etc\hosts | Select-String api.local
```

You should see `127.0.0.1 api.local`. If not, follow
[Hosts file setup](Installation#hosts-file-setup). If you just added the entry,
flush the DNS cache:

```bash
# macOS
sudo dscacheutil -flushcache && sudo killall -HUP mDNSResponder

# Linux with systemd-resolved
sudo systemctl restart systemd-resolved
```

```cmd
:: Windows
ipconfig /flushdns
```

## "host not mapped" error page

UNCORS received the request but no mapping on that port matches its host name.
Check that the host in the browser's address bar is spelled exactly like the
host in `from`. A `{name}` placeholder matches a single label, so
`{name}.local.com` matches `api.local.com` but not `v2.api.local.com`.

## HTTPS certificate errors

The browser shows `NET::ERR_CERT_AUTHORITY_INVALID` or a similar error, or
`curl` reports "SSL certificate problem".

If UNCORS refuses to start with "requires a local CA certificate", create the
CA:

```bash
uncors generate-certs
```

This creates `~/.config/uncors/ca.crt` and `~/.config/uncors/ca.key`.

If the browser still rejects the certificate, the CA is not trusted yet.

macOS:

```bash
open ~/.config/uncors/ca.crt
# In Keychain Access, set the certificate to "Always Trust"
```

Linux (Debian and Ubuntu):

```bash
sudo cp ~/.config/uncors/ca.crt /usr/local/share/ca-certificates/uncors-ca.crt
sudo update-ca-certificates
```

Windows (Command Prompt):

```cmd
certutil -addstore -user Root %USERPROFILE%\.config\uncors\ca.crt
```

Firefox has its own certificate store. Open Settings, then Privacy & Security,
then Certificates, View Certificates, and import `ca.crt` on the Authorities
tab.

If the CA has expired, UNCORS cannot sign new certificates. Check the dates and
create a new CA, then trust it again:

```bash
openssl x509 -in ~/.config/uncors/ca.crt -noout -dates
uncors generate-certs --force
```

Connect by host name, not by IP address. UNCORS picks the certificate from the
host name the client sends (SNI), and a client that connects to an IP address
sends none.

To test without trusting the CA, `curl -k https://api.local:8443/` skips
certificate checks.

## CORS errors still appear

The browser console still reports CORS errors.

Check that the request goes through UNCORS. Every handled request is printed
in the UNCORS console. If the request is missing, your app is probably still
calling the original domain instead of the local one.

Check preflight handling. UNCORS answers `OPTIONS` requests itself unless
`options-handling.disabled` is `true`. With handling disabled, the upstream
server must answer preflight requests correctly:

```yaml
options-handling:
  disabled: false # default; UNCORS answers preflight requests
```

Check custom headers. Headers you set in mocks, scripts, or
`options-handling.headers` replace the CORS headers UNCORS adds. A
hand-written `Access-Control-Allow-Origin: *` breaks requests sent with
credentials, which need the exact origin. Remove the custom header to get the
default behavior back.

Check static files. Files served from a [static directory](Static-File-Serving)
come without CORS headers. Load them from the same origin as the page, or serve
them through a mapping without `statics`.

Clear the browser cache, or try a private window, in case the browser kept an
old preflight result.

## Configuration problems

UNCORS validates the configuration at startup and prints every error with the
path of the bad value, for example
`mappings[0].mocks[0].response.code code must be in range 100-599`.

"mappings must not be empty" means no mapping was found. Check the
`--config` path, and use an absolute path if you start UNCORS from another
directory:

```bash
uncors --config /absolute/path/to/.uncors.yaml
```

For YAML syntax errors, the message includes the line number. Common causes are
tabs used for indentation, a missing colon after a key, and unquoted values
that contain `:` or `#`.

"directory does not exist" or "does not exist" for a path in `statics`,
`mocks`, or `scripts` usually means a relative path or `~`. Paths are relative
to the directory UNCORS was started from, and `~` is not expanded.

"cannot unmarshal !!str `1m 30s` into time.Duration" means a duration contains
a space. Write `1m30s` instead.

UNCORS reloads the configuration file when it changes. If the new version is
invalid, UNCORS prints the error and keeps the previous configuration until you
fix the file.

## Mocks don't respond

The request reaches the upstream server instead of the mock.

The path must match exactly. `/api/users` does not match `/api/users/` or
`/api/users/1`. Use `{name}` segments for variable parts:

```yaml
mocks:
  - path: /api/users/{id}
    response:
      code: 200
      raw: '{"id": "123"}'
```

A `method`, `queries`, or `headers` filter must match too:

```yaml
mocks:
  - path: /api/users
    method: POST # GET requests go to the upstream server
    response:
      code: 201
      raw: created
```

A [static directory](Static-File-Serving) whose `path` covers the mock's path
handles the request first. With `index` set, it answers every request under
its path.

A [rewrite](Request-Rewriting) sends the request straight to the upstream
server; mocks only see the original path.

If the mock answers but with the wrong status code, check whether it uses
`file`. File responses are currently always sent with status 200.

## Static files are not served

Check the directory path, relative to where you started UNCORS:

```bash
ls -la ./dist
```

Check the URL prefix. With this configuration the URL must start with
`/assets`:

```yaml
statics:
  - path: /assets
    dir: ./dist
```

```bash
curl http://app.local:3000/assets/style.css # served from ./dist/style.css
curl http://app.local:3000/style.css        # proxied upstream
```

For client-side routing, set `index` so unknown paths return the app:

```yaml
statics:
  - path: /
    dir: ./build
    index: index.html
```

## Scripts don't run or fail

If the upstream server answers instead of the script, check `path`, `method`,
and the other filters the same way as for mocks.

If the response is a 500 error, the script failed. The Lua error and line
number are printed in the UNCORS console. Common causes are reading a field of
`nil`, for example after `json.decode` failed, and concatenating `nil` with
`..`.

If headers set in the script are missing, they were set after
`response:WriteHeader()` or after the first write. Set headers first.

## Rewrites go to the wrong place

A rewrite replaces the whole path with `to` and drops the query string. A
`{name}` segment captures one path segment only. If the upstream server needs
more of the path, capture each segment in `from` and use it in `to`.

A rewrite with `host` uses the scheme of the incoming request. From an `http://`
source, the rewritten host is called over plain HTTP.

## Proxy problems

Upstream requests fail with proxy errors.

The `proxy` option needs a full URL with a scheme:

```yaml
proxy: http://proxy.example.com:8080 # correct
# proxy: proxy.example.com:8080      # rejected: "proxy is not a valid URL"
```

Test the proxy itself:

```bash
curl -x http://proxy.example.com:8080 https://example.com
```

Without `proxy`, UNCORS uses `HTTP_PROXY`, `HTTPS_PROXY`, and `NO_PROXY` from
the environment. If those point to a proxy you don't want, unset them before
starting UNCORS:

```bash
unset HTTP_PROXY HTTPS_PROXY http_proxy https_proxy
```

For a proxy with authentication, include the credentials in the URL:

```yaml
proxy: http://username:password@proxy.example.com:8080
```

## Slow responses or high memory use

Slow upstream responses can be cached:

```yaml
cache-config:
  expiration-time: 10m

mappings:
  - from: http://api.local:3000
    to: https://api.example.com
    cache:
      - /api/**
```

To compare, time the upstream server directly:

```bash
time curl https://api.example.com/endpoint
```

If memory use is high, lower `cache-config.max-size` (100 MB by default) or
`expiration-time`, or narrow the `cache` globs so large responses aren't
cached:

```yaml
cache-config:
  expiration-time: 5m
  max-size: 52428800 # 50 MB
```

HAR recording keeps all entries of the current run in memory. For long
sessions with heavy traffic, turn it off when you don't need it.

## Getting more help

Capture diagnostic logs:

```bash
UNCORS_LOGGING=uncors.log uncors --config .uncors.yaml
```

Check your version:

```bash
uncors --version
```

If the problem remains, open an issue on
[GitHub](https://github.com/evg4b/uncors/issues) with the UNCORS version, your
operating system, the configuration file with secrets removed, the diagnostic
log, and the steps to reproduce.
