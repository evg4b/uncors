The HAR collector writes the traffic of a mapping to an
[HTTP Archive (HAR 1.2)](https://w3c.github.io/web-performance/specs/HAR/Overview.html)
file. Open it in browser DevTools or another HAR viewer to see what your app
sent and what the server answered.

## Configuration

The short form is the path of the output file:

```yaml
mappings:
  - from: http://api.local:3000
    to: https://api.example.com
    har: ./recordings/api.har
```

The object form adds one option:

```yaml
mappings:
  - from: http://api.local:3000
    to: https://api.example.com
    har:
      file: ./recordings/api.har
      capture-secure-headers: false
```

| Option                   | Type    | Default | Description                                                                       |
| ------------------------ | ------- | ------- | --------------------------------------------------------------------------------- |
| `file`                   | string  | -       | Output file. It must have an extension, such as `.har`. Empty means no recording. |
| `capture-secure-headers` | boolean | `false` | Keep the headers listed under [Secure headers](#secure-headers) in the recording. |

The path is relative to the directory you start UNCORS from, and missing
directories are created. Each mapping needs its own file; two mappings cannot
share one.

## What is recorded

Only requests that reach the proxy path are recorded:

| Request                                                                           | Recorded |
| --------------------------------------------------------------------------------- | -------- |
| Proxied to the upstream server, including [rewritten](Request-Rewriting) requests | Yes      |
| Answered from the [cache](Response-Caching)                                       | Yes      |
| `OPTIONS` preflight answered by UNCORS                                            | Yes      |
| Under a [static](Static-File-Serving) path but not found, so proxied              | Yes      |
| Served from a static directory                                                    | No       |
| Answered by a [mock](Response-Mocking) or a [script](Script-Handler)              | No       |
| WebSocket or another protocol upgrade                                             | No       |

Each entry has the request and response headers, cookies, query string,
bodies, and timings. Response bodies are stored decompressed (`gzip`,
`deflate`, `br`, and `zstd` are supported); binary bodies are stored in base64.

> [!NOTE]
> A request body over 10 MiB is recorded by its size only. A response body
> over 10 MiB is cut to its first 10 MiB, and `content.comment` says so; the
> client still gets all of it. A response that would decode to more than 50 MiB
> is stored as received, in base64.

## Secure headers

By default these headers are left out of every entry, and the `cookies` lists
are empty:

- `Cookie` and `Set-Cookie`
- `Authorization` and `WWW-Authenticate`
- `Proxy-Authorization` and `Proxy-Authenticate`
- `Api-Key`, and names ending in `-Api-Key`, `-Apikey`, `-Token`, or `-Secret`
- names starting with `X-Auth-`

Set `capture-secure-headers: true` to keep them.

> [!WARNING]
> URLs, query strings, and bodies are always recorded in full, so a token in the
> URL or in a login response ends up in the file. With
> `capture-secure-headers: true`, cookies and tokens are written in plain text
> too. Don't commit such files.

## When the file is written

Recording never slows down requests. A background writer updates the file
about once a second, through a temporary file, so the `.har` file is always
complete and valid even while UNCORS runs. It is written a last time when
UNCORS stops.

A recording covers one run of one configuration. When UNCORS starts, and when
it reloads the configuration, the recording starts empty and replaces the
existing file. Copy the file first if you want to keep an earlier session.

> [!NOTE]
> Entries wait in a queue of 4,096. If it fills up during a burst of traffic,
> new entries are dropped rather than delaying requests.

## Viewing HAR files

- Chrome, Edge, and Firefox: DevTools, Network tab, Import HAR.
- Postman: File, Import, then select the `.har` file.
- Online: [HAR Viewer](https://google.github.io/har-viewer/).
