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
share one:

```text
mappings[1].har: HAR file "./api.har" is already used by mappings[0].har
```

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

<details>
<summary>Entry format and body details</summary>

Each entry has the request and response headers, cookies, query string,
bodies, and timings. `httpVersion` is the protocol the client used, such as
`HTTP/1.1` or `HTTP/2.0`, for both the request and the response.

Entries are sorted by `startedDateTime`, oldest first. `url` never includes the
fragment, `queryString` keeps the order the parameters were sent in, and
headers are sorted by name. Response cookies keep `path`, `domain`, `expires`,
`httpOnly`, and `secure`. `creator` is `uncors` and the version that wrote the
file.

#### Request bodies

A request with a body gets a `postData` object, as described in HAR 1.2:

| Field      | Value                                                                                                 |
| ---------- | ----------------------------------------------------------------------------------------------------- |
| `mimeType` | The request's `Content-Type` header                                                                   |
| `params`   | The parameters of an `application/x-www-form-urlencoded` body, in order                               |
| `text`     | Any other body, if it is valid UTF-8. Left out for forms, as HAR 1.2 requires, and for binary bodies. |

`bodySize` is the size of the body in bytes, so a binary upload such as an
image is recorded by its size only. Requests without a body have no `postData`.

UNCORS reads up to 10 MiB of the body before sending the request upstream. If
the first 10 MiB cannot be read, for example because the client disconnected
in the middle of an upload, the request fails and is not sent to the upstream
server, so the upstream never gets a truncated body.

A body larger than 10 MiB is streamed to the upstream server instead of being
held in memory. Its entry has a `postData` with only `mimeType`, and `bodySize`
is the number of bytes the upstream read.

#### Response bodies

Response bodies are stored decompressed. UNCORS decodes these
`Content-Encoding` values: `gzip`, `deflate`, `br` (Brotli), and `zstd`.
Browsers usually ask for Brotli or zstd, and UNCORS forwards the browser's
`Accept-Encoding`, so most responses arrive compressed.

| Body                                             | `text`                          | `encoding` |
| ------------------------------------------------ | ------------------------------- | ---------- |
| Valid UTF-8 after decoding                       | The body                        | -          |
| Binary after decoding, such as an image          | Base64 of the decoded body      | `base64`   |
| Unknown encoding, or a body that fails to decode | Base64 of the bytes as received | `base64`   |

A body that would decode to more than 50 MiB is not decoded. It is stored as
received, in base64, and `content.comment` says why.

`content.size` is the size of the decoded body, and `bodySize` is the size as
it came over the wire. For a decoded compressed body, `content.compression` is
the number of bytes saved.

Only the first 10 MiB of a response body are recorded; the client still gets
the whole response. For a longer body, `bodySize` is the full size,
`content.text` holds the first 10 MiB, and `content.comment` says it was cut.

</details>

## Secure headers

By default these headers are left out of every entry, and the `cookies` lists
are empty:

- `Cookie` and `Set-Cookie`
- `Authorization` and `WWW-Authenticate`
- `Proxy-Authorization` and `Proxy-Authenticate`
- `Api-Key`, and names ending in `-Api-Key`, `-Apikey`, `-Token`, or `-Secret`
- names starting with `X-Auth-`

Names are compared without regard to case. Set `capture-secure-headers: true` to keep them.

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
existing file. Requests still running during a reload are not recorded. Copy
the file first if you want to keep an earlier session.

<details>
<summary>How the file is written</summary>

Recording never slows down requests. Entries go to a queue, and a background
writer adds them to the file at most once a second, so a new request shows up
in the file within about a second. Each write replaces the whole file, so with
a large recording the writer waits at least ten times as long as the last write
took, which keeps writing to about a tenth of the time. Entries already written
are kept on disk rather than in memory, in a hidden file next to the `.har`
file (for example `.api.har.1234.spool`) that is removed when the recording
ends. It writes to a temporary file next to the `.har` file and renames it over
the old one, so the `.har` file is always complete and valid, even if you open
it while UNCORS runs. The file is written once more when UNCORS stops, or when a
reload moves recording to another file or turns it off. A configuration that
fails to start, or a reload that fails and keeps the previous configuration
running, leaves the file untouched.

</details>

> [!NOTE]
> Entries wait in a queue of 4,096. If it fills up during a burst of traffic,
> new entries are dropped rather than delaying requests.

## Viewing HAR files

- Chrome, Edge, and Firefox: DevTools, Network tab, Import HAR.
- Postman: File, Import, then select the `.har` file.
- Online: [HAR Viewer](https://google.github.io/har-viewer/).
