Common problems with UNCORS and how to fix them.

## First checks

- UNCORS is running and printed no errors at startup.
- The hosts file maps your local domain to `127.0.0.1`.
- The URL you open uses the same host and port as the mapping's `from`.
- Your browser or HTTP client isn't sending local traffic through a proxy.
- The failing request shows up in the UNCORS console. If it doesn't, it never
  reached UNCORS.

## Reading the request log

Every handled request is printed as one line:

```text
 CACHE  200 GET    http://api.local:3000/users
 PROXY  200 GET    http://api.local:3000/old → /new
 PROXY  502 GET    http://api.local:3000/orders
  ↳ failed to execute request: dial tcp 10.0.0.5:443: connect: connection refused
```

The label names what produced the response:

| Label   | Response came from                                          |
| ------- | ----------------------------------------------------------- |
| `PROXY` | The upstream server (`to`), including cache misses.         |
| `CACHE` | The response cache. The first request for a URL is `PROXY`. |
| `MOCK`  | A mock.                                                     |
| `SCRPT` | A Lua script.                                               |
| `STATC` | A static directory.                                         |
| `OPTNS` | UNCORS's own preflight answer.                              |

A line without a label is a request no mapping took, for example one for a host
that isn't mapped. `→` shows the path a rewrite sent the request to. A request
that failed in UNCORS gets a second line with the error.

## Error pages

When a request fails in UNCORS, the client gets a plain-text error page with
the error message. Its status tells what went wrong:

| Status | Cause                                                                 |
| ------ | --------------------------------------------------------------------- |
| `404`  | No mapping on that port matches the host.                             |
| `502`  | UNCORS could not reach the upstream server, or the connection failed. |
| `504`  | The upstream server or its connection timed out.                      |
| `500`  | Anything else, such as a failed script, a missing file, or a crash.   |

The error page carries CORS headers, so the browser shows its status and text
instead of a CORS error.

> [!NOTE]
> If the upstream server drops the connection after the response has started,
> there is no error page: UNCORS closes the connection and the client sees a
> network error.

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

UNCORS received the request but no mapping on that port matches its host name,
so it answered with status 404. The message names the host it got, such as
`host not mapped: api.local:3000`. Check that the host in the browser's address
bar is spelled exactly like the host in `from`. A `{name}` placeholder matches
a single label, so `{name}.local.com` matches `api.local.com` but not
`v2.api.local.com`.

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

If the CA has expired (UNCORS warns a week before), check the dates, create a
new CA, trust it, and restart UNCORS:

```bash
openssl x509 -in ~/.config/uncors/ca.crt -noout -dates
uncors generate-certs --force
```

If UNCORS shows "TLS: no https mapping for host", the client asked for a host
name that no `https://` mapping covers. Add a mapping for that host. Connect by
host name, not by IP address. See
[HTTPS configuration](Configuration#https-configuration).

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

Check custom headers. Headers set in mocks, scripts, or
`options-handling.headers` replace the CORS headers UNCORS adds. A
hand-written `Access-Control-Allow-Origin: *` breaks requests sent with
credentials, which need the exact origin.

Check static files. Files from a [static directory](Static-File-Serving) come
without CORS headers.

Check `allowed-origins`. A page whose origin matches none of the patterns gets
responses without CORS headers, and its preflight requests get `403`. See
[Allowed origins](Configuration#allowed-origins).

Clear the browser cache, or try a private window, in case the browser kept an
old preflight result.

## Any website can read responses through UNCORS

Without `allowed-origins`, UNCORS copies any `Origin` into
`Access-Control-Allow-Origin` and allows credentials. A page from any site you
open while UNCORS runs can send requests to the mapping, with your cookies for
the local host, and read the responses. That includes upstream data that those
cookies authorize, because UNCORS forwards the `Cookie` header. Listening on
`127.0.0.1` does not prevent this: the requests come from your own browser.

Limit CORS headers to the origins of your own frontend:

```yaml
mappings:
  - from: http://api.local:3000
    to: https://api.example.com
    allowed-origins:
      - http://localhost:*
      - http://app.local:3000
```

> [!NOTE]
> Requests from other origins are still forwarded. This stops a page from
> reading the answers, not from sending simple requests (such as a form `POST`)
> that change data.

## Configuration problems

UNCORS validates the configuration at startup and prints every error with the
path of the bad value, for example
`mappings[0].mocks[0].response.code must be in range 200-599`.

"mappings must not be empty" means no mapping was found, or the file is empty.
Check the `--config` path, and use an absolute path if you start UNCORS from
another directory:

```bash
uncors --config /absolute/path/to/.uncors.yaml
```

For YAML syntax errors, the message includes the line number. Common causes are
tabs used for indentation, a missing colon after a key, and unquoted values
that contain `:` or `#`.

"directory does not exist" or "does not exist" for a path in `statics`,
`mocks`, or `scripts` usually means a relative path or `~`. Paths are relative
to the directory UNCORS was started from, and `~` is not expanded.

A value of the wrong type is reported with its path and line, for example
`mappings[0].mocks[0].response.delay must be a duration without spaces, such as 500ms or 1m30s, got "1m 30s" (line 10)`.
Write durations without spaces (`1m30s`), lists as YAML lists, and numbers
without quotes.

"unknown field" means a key is misspelled or is not an option at that place in
the file. The error names the path and the line of the key.

"port N is already used for http" means two mappings use the same port with
different schemes. Give the `https://` mapping its own port.

If a reloaded file is invalid, or one of its ports is in use, UNCORS prints
the error and keeps the previous configuration.

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
number are shown under the request in the UNCORS console and in the response
body. Common causes are reading a field of `nil`, for example after
`json.decode` failed, and concatenating `nil` with `..`.

If headers set in the script are missing, they were set after
`response:WriteHeader()` or after the first write. Set headers first.

## Rewrites go to the wrong place

A rewrite replaces the whole path with `to` and keeps the query string. A
`{name}` segment captures one path segment only. If the upstream server needs
more of the path, capture each segment in `from` and use it in `to`.

A rewrite `host` without a scheme uses the scheme of the incoming request, so
from an `http://` source it is called over plain HTTP. Write the scheme, for
example `host: https://auth.example.com`.

## WebSocket connections fail

UNCORS proxies WebSocket connections on every mapping. Connect with `ws://` to
an `http` mapping and with `wss://` to an `https` mapping; `wss://` needs the
same trusted certificate as HTTPS requests (see
[HTTPS certificate errors](#https-certificate-errors)).

A working connection shows in the request log as `PROXY` with status `101`.
Another status means the upstream server refused the handshake: check its logs
and the `Origin` it received (UNCORS rewrites it to the upstream host). `MOCK`
or `SCRPT` means a mock or a script matches the WebSocket path.

Open connections are closed when UNCORS stops or reloads its configuration,
for example after the configuration file changes. Clients have to reconnect.

## Proxy problems

Upstream requests fail with proxy errors. The cause is shown under the request
in the UNCORS console and on the error page UNCORS returns.

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

If memory use is high, lower `cache-config.max-size` (100 MiB by default) or
`expiration-time`, or narrow the `cache` globs:

```yaml
cache-config:
  expiration-time: 5m
  max-size: 52428800 # 50 MiB
```

## Getting more help

Capture traces and diagnostic logs with a local collector (see
[Tracing and diagnostics](Configuration#tracing-and-diagnostics)):

```bash
docker run --rm -p 16686:16686 -p 4318:4318 jaegertracing/jaeger
UNCORS_TRACE=true OTEL_LOGS_EXPORTER=none OTEL_METRICS_EXPORTER=none uncors --config .uncors.yaml
```

Check your version:

```bash
uncors --version
```

If the problem remains, open an issue on
[GitHub](https://github.com/evg4b/uncors/issues) with the UNCORS version, your
operating system, the configuration file with secrets removed, the failing
trace if you captured one, and the steps to reproduce.
