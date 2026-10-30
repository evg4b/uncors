# har

Reads, builds, and writes
[HAR 1.2](https://w3c.github.io/web-performance/specs/HAR/Overview.html)
archives. The package imports nothing from the rest of uncors.

## Archives

```go
archive := har.New(har.Creator{Name: "my-tool", Version: "1.0"}, entries...)
err := har.Encode(w, archive)
archive, err := har.Decode(r)
```

The types cover every field of the spec. Empty optional fields are left out,
and optional numbers whose zero means something (`compression`, the optional
timings, the page timings) are pointers. `Decode` skips a byte order mark and
ignores unknown fields.

## Entries from net/http

```go
entry := har.NewEntry(start, elapsed,
    har.NewRequest(req, requestBody),
    har.NewResponse(resp, responseBody),
)
```

- Response bodies are decoded by `Content-Encoding` (`gzip`, `deflate`, `br`,
  `zstd`). A body that is not UTF-8 or fails to decode is stored as base64.
  A body that would decode past `MaxDecodedBodySize` (50 MiB) is stored as
  received, with a comment.
- URL-encoded forms go to `postData.params`, other UTF-8 bodies to
  `postData.text`.
- Request URLs never include the fragment. Query parameters keep their order,
  headers are sorted by name, and response cookies keep their attributes.
- `Cookie`, `Set-Cookie`, `Authorization`, `WWW-Authenticate`, the two proxy
  auth headers, API key and token headers (see `IsSecureHeader`), and all
  cookies are left out unless you pass `har.WithSecureHeaders(true)`.

`NewHeaders`, `NewQueryString`, `NewCookies`, `NewPostData`, `NewContent`, and
`DecodeBody` build the individual parts.

## Writing a file

```go
writer := har.NewWriter("traffic.har",
    har.WithFlushHandler(func(f har.Flush) { /* f.Err, f.Entries, f.Bytes */ }),
)
defer writer.Close()

writer.AddEntry(entry)
```

`AddEntry` never blocks: it returns `false` and drops the entry when the queue
is full. A background goroutine keeps the entries sorted by `startedDateTime`
and rewrites the file at most once per flush interval, or once per ten times
the last write's duration when that is longer, through a temporary file and a
rename, so the file on disk is always a valid archive. Written entries are kept
in a hidden spool file next to the archive (`.traffic.har.*.spool`), not in
memory, and each write copies them from there. `Close` writes what is left,
and `Discard` stops without writing; both remove the spool file. The flush and
error handlers run on that goroutine.

The other options are `WithFileSystem` (the OS file system by default),
`WithCreator`, `WithQueueSize`, `WithFlushInterval`, and `WithErrorHandler`.
