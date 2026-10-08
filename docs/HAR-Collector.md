The HAR collector writes the traffic of a mapping to an
[HTTP Archive (HAR 1.2)](https://w3c.github.io/web-performance/specs/HAR/Overview.html)
file. You can open the file in browser DevTools or other HAR viewers to see
what your app sent and what the server answered, or share it with someone
debugging the same problem.

## Quick start

Add `har` to a mapping with the path of the output file:

```yaml
mappings:
  - from: http://api.local:3000
    to: https://api.example.com
    har: ./recordings/api.har
```

## Configuration

The short form is a file path. The object form adds one option:

```yaml
mappings:
  - from: http://api.local:3000
    to: https://api.example.com
    har:
      file: ./recordings/api.har
      capture-secure-headers: false
```

| Option                   | Type    | Default | Description                                                                         |
| ------------------------ | ------- | ------- | ----------------------------------------------------------------------------------- |
| `file`                   | string  | -       | Output file. It must have an extension, such as `.har`. Empty means no recording.   |
| `capture-secure-headers` | boolean | `false` | Keep the headers listed under [Secure headers](#secure-headers) in the recording.  |

The path is relative to the directory you start UNCORS from. Missing parent
directories are created.

## What is recorded

The collector sees requests that UNCORS sends to the upstream server, plus the
responses it produces on that path:

| Request                                                    | Recorded |
| ---------------------------------------------------------- | -------- |
| Proxied to the upstream server                             | Yes      |
| Answered from the [cache](Response-Caching)                | Yes      |
| `OPTIONS` preflight answered by UNCORS                     | Yes      |
| Under a [static](Static-File-Serving) path, file not found, proxied | Yes |
| [Rewritten](Request-Rewriting) and proxied                 | Yes      |
| Answered by a [mock](Response-Mocking)                     | No       |
| Answered by a [script](Script-Handler)                     | No       |
| Served from a static directory                             | No       |

Each entry has the request and response headers, cookies, query string,
bodies, and timings. Bodies compressed with gzip or deflate are stored
decompressed. Bodies with other encodings, such as Brotli, are stored as
base64.

## Secure headers

To keep credentials off the disk, these headers are left out of every entry by
default, and so are the request and response cookie lists:

| Header                | Why it is sensitive                          |
| --------------------- | -------------------------------------------- |
| `Cookie`              | Session identifiers sent by the browser      |
| `Set-Cookie`          | Session identifiers set by the server        |
| `Authorization`       | Bearer tokens and Basic auth credentials     |
| `WWW-Authenticate`    | Auth challenges, which reveal scheme and realm |
| `Proxy-Authorization` | Proxy credentials                            |
| `Proxy-Authenticate`  | Proxy auth challenges                        |

Set `capture-secure-headers: true` to keep them:

```yaml
mappings:
  - from: http://api.local:3000
    to: https://api.example.com
    har:
      file: ./recordings/auth-debug.har
      capture-secure-headers: true
```

> [!WARNING]
> Such a file contains tokens and cookies in plain text. Don't commit it or
> share it without removing them, and delete it when you are done.

## File lifecycle

Each mapping writes its own file, so traffic from different mappings never
mixes:

```yaml
mappings:
  - from: http://api.local:3000
    to: https://api.example.com
    har: ./recordings/api.har

  - from: http://auth.local:3001
    to: https://auth.example.com
    har: ./recordings/auth.har
```

Recording never slows down requests. Entries go to a queue, and a background
writer rewrites the whole file after each batch of new entries. It writes to a
temporary file and renames it over the old one, so the `.har` file is always
complete and valid, even if you open it while UNCORS runs. The file is written
once more when UNCORS stops.

A recording covers one run of one configuration. When UNCORS starts, and when
it reloads a changed configuration file, the recording starts empty and the
next write replaces the existing file. Copy the file first if you want to keep
an earlier session.

> [!NOTE]
> The queue holds 4,096 entries. If it fills up during a burst of traffic, new
> entries are dropped rather than delaying requests. This is rare in normal
> development.

## Viewing HAR files

| Tool                   | How                                                                 |
| ---------------------- | ------------------------------------------------------------------- |
| Chrome or Edge DevTools | Network tab, then Import HAR                                       |
| Firefox DevTools       | Network tab, then Import HAR                                        |
| Postman                | File, Import, then select the `.har` file                           |
| HAR Viewer (online)    | [google.github.io/har-viewer](https://google.github.io/har-viewer/) |

## Example: recording next to other features

```yaml
mappings:
  - from: http://app.local:3000
    to: https://api.example.com
    har: ./recordings/app.har
    cache:
      - /api/config
    mocks:
      - path: /api/feature-flags
        response:
          code: 200
          headers:
            Content-Type: application/json
          raw: '{"newUi": true}'
```

`/api/config` is recorded both when it is fetched and when it comes from the
cache. `/api/feature-flags` is answered by the mock and does not appear in the
file.
