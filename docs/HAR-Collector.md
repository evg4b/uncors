The HAR Collector records every request and response that passes through a
mapping to an [HTTP Archive (HAR
1.2)](https://w3c.github.io/web-performance/specs/HAR/Overview.html) file. The
resulting file can be opened in browser DevTools, Postman, or any HAR-compatible
viewer for offline inspection, debugging, or sharing with teammates.

**Benefits:**

 - Capture real API traffic without touching the browser or adding custom
   scripts
 - Replay or inspect captured traffic in any HAR-compatible tool
 - Share exact request/response sequences as a single portable file
 - Audit what headers and payloads your frontend actually sends

## Quick Start

Add `har` to any mapping with the path to the output file:

```yaml
mappings:
  - from: http://api.local:3000
    to: https://api.example.com
    har: ./recordings/api.har
```

UNCORS creates the file on the first request and refreshes it as you browse.

## Configuration

### Shorthand Syntax

Pass a file path string directly - the most concise form:

```yaml
mappings:
  - from: http://api.local:3000
    to: https://api.example.com
    har: ./recordings/api.har
```

### Full Syntax

Use the object form when you need extra control:

```yaml
mappings:
  - from: http://api.local:3000
    to: https://api.example.com
    har:
      file: ./recordings/api.har
      capture-secure-headers: false
```

### Configuration Properties

| Property                 | Type    | Default | Description                                                                                  |
| ------------------------ | ------- | ------- | -------------------------------------------------------------------------------------------- |
| `file`                   | string  | -       | Path to the output `.har` file. Collector is disabled when this is empty.                    |
| `capture-secure-headers` | boolean | `false` | Include security-sensitive headers (see [Secure Headers](#secure-headers)) in the HAR entry. |

## Secure Headers

To prevent credentials from being written to disk, the following headers are
**stripped from HAR entries by default**:

| Header                | Why it is sensitive                            |
| --------------------- | ---------------------------------------------- |
| `Cookie`              | Session identifiers sent by the browser        |
| `Set-Cookie`          | Session identifiers set by the upstream server |
| `Authorization`       | Bearer tokens, Basic auth credentials          |
| `WWW-Authenticate`    | Server auth challenges (reveals scheme/realm)  |
| `Proxy-Authorization` | Proxy credentials                              |
| `Proxy-Authenticate`  | Proxy auth challenges                          |

Set `capture-secure-headers: true` to include these headers in the recording:

```yaml
mappings:
  - from: http://api.local:3000
    to: https://api.example.com
    har:
      file: ./recordings/api.har
      capture-secure-headers: true
```

> [!WARNING]
> HAR files with `capture-secure-headers: true` contain tokens, cookies, and other
> credentials in plain text. Do not commit them to version control or share them
> without scrubbing sensitive values first.

## Per-Mapping Isolation

Each mapping writes to its own independent HAR file. Traffic from different
mappings never mixes:

```yaml
mappings:
  - from: http://api.local:3000
    to: https://api.example.com
    har: ./recordings/api.har        # captures api.local traffic only

  - from: http://auth.local:3001
    to: https://auth.example.com
    har: ./recordings/auth.har       # captures auth.local traffic only
```

## File Lifecycle

 - **Created** on the first captured request; parent directories are created
   for you.
 - **Recorded continuously** into a `<name>.har.jsonl` journal next to the
   archive. Appending one entry costs the same whether it is the first or the
   ten-thousandth, so a long session does not get slower.
 - **Rebuilt** from that journal at most a few times per second, written to a
   temporary file and renamed over the target. The `.har` file is therefore
   always valid and complete if you open it mid-session, though it may be a
   fraction of a second behind the newest request.
 - **Finalised and closed** on shutdown and when the configuration is reloaded:
   every pending entry is written, and the journal is removed.
 - **Bounded** at 100,000 entries per archive. Beyond that, entries are dropped
   and a warning is logged rather than letting one session grow without limit.

> [!TIP]
> A `.har.jsonl` file left next to your archive means the process was killed
> before it could finalise the recording. It holds one JSON entry per line and
> can be recovered by hand.

> [!NOTE]
> If the internal write buffer (4,096 entries) fills up during a traffic spike,
> new entries are silently dropped rather than slowing down your requests. This is
> uncommon in normal development use.

## Viewing Captured HAR Files

Open the generated file with any of these tools:

| Tool                       | How                                                                 |
| -------------------------- | ------------------------------------------------------------------- |
| **Chrome / Edge DevTools** | Network tab → Import HAR                                            |
| **Firefox DevTools**       | Network tab → Import HAR                                            |
| **Postman**                | File → Import → select `.har`                                       |
| **HAR Viewer** (online)    | [google.github.io/har-viewer](https://google.github.io/har-viewer/) |

## Examples

### Record All API Traffic

```yaml
mappings:
  - from: http://api.local:3000
    to: https://api.example.com
    har: ./recordings/session.har
```

### Debug Authentication Flows

Capture auth headers to see exactly what the browser sends:

```yaml
mappings:
  - from: http://api.local:3000
    to: https://api.example.com
    har:
      file: ./recordings/auth-debug.har
      capture-secure-headers: true
```

> [!WARNING]
> Delete `auth-debug.har` when done - it contains your credentials in plain text.

### Combine with Other Features

HAR recording works alongside mocking, caching, and static file serving:

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
          raw: '{"newUi": true}'
    statics:
      - path: /
        dir: ./dist
        index: index.html
```
