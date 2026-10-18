# har

Reads, builds, and writes [HAR 1.2](https://w3c.github.io/web-performance/specs/HAR/Overview.html)
archives. It has no dependencies on the rest of uncors, so it can be used on its
own.

## API

### Types

`HAR`, `Log`, `Creator`, `Browser`, `Page`, `PageTimings`, `Entry`, `Request`,
`Response`, `Cookie`, `NameValue`, `PostData`, `Param`, `Content`, `Cache`,
`CacheState`, and `Timings` mirror the HAR 1.2 objects with all their fields,
required and optional, and marshal to the field names of the spec. Optional
fields are left out when empty; optional numbers whose zero is meaningful
(`compression`, `blocked`, `dns`, `connect`, `ssl`, `onContentLoad`, `onLoad`)
are pointers. Every entry has a `cache` object, `{}` when nothing is known.

### Archives

```go
archive := har.New(har.Creator{Name: "my-tool", Version: "1.0"}, entries...)

err := har.Encode(w, archive)   // indented UTF-8 JSON with a trailing newline
archive, err := har.Decode(r)   // ignores a byte order mark and custom fields
```

`har.Version` is the format version, `1.2`.

### Building entries from net/http

```go
request := har.NewRequest(req, requestBody)
response := har.NewResponse(resp, responseBody)
entry := har.NewEntry(start, elapsed, request, response)
```

- `NewRequest` takes the URL from `req.Host` and `req.RequestURI` for a server
  request (`https` when `req.TLS` is set), from the request target itself when
  it is in absolute form, and from `req.URL` for a client request. The fragment
  is never included.
- `NewResponse` decodes the body by `Content-Encoding` (`gzip`, `deflate`, `br`,
  `zstd`) and stores it as text, or as base64 when it is not UTF-8 or cannot be
  decoded. `BodySize` is the size of the body as sent, `Content.Size` the
  decoded size, and `Content.Compression` the difference.
- `NewPostData` lists an `application/x-www-form-urlencoded` body in `params`
  and any other UTF-8 body in `text`; HAR 1.2 makes the two mutually exclusive.
- Headers are sorted by name, query parameters keep their order, and response
  cookies keep `path`, `domain`, `expires`, `httpOnly`, and `secure`.
- `NewEntry` records the elapsed time as `wait`, so `time` equals the sum of the
  timings.
- `Cookie`, `Set-Cookie`, `Authorization`, `WWW-Authenticate`,
  `Proxy-Authorization`, and `Proxy-Authenticate` headers, and all cookies, are
  left out unless `har.WithSecureHeaders(true)` is passed. `har.IsSecureHeader`
  reports whether a header is one of them.

Smaller pieces are exported too:

```go
har.NewHeaders(header, opts...)       // []NameValue
har.NewQueryString(rawQuery)          // []NameValue in query order
har.NewCookies(cookies)               // []Cookie
har.NewPostData(body, mimeType)       // nil for an empty body
har.NewContent(body, encoding, mime)  // decoded body as Content
har.DecodeBody(body, encoding)        // decoded bytes, or ErrUnsupportedEncoding
```

### Writing a file as traffic arrives

```go
writer := har.NewWriter("traffic.har",
    har.WithFileSystem(fs),                  // afero.Fs, the OS file system by default
    har.WithCreator(creator),                // DefaultCreator by default
    har.WithQueueSize(4096),                 // DefaultQueueSize
    har.WithFlushInterval(time.Second),      // DefaultFlushInterval
    har.WithFlushHandler(func(f har.Flush) { /* f.Err, f.Entries, f.Bytes, ... */ }),
    har.WithErrorHandler(func(err error) { /* errors.Is(err, har.ErrEncodeEntry) */ }),
)
defer writer.Close()

ok := writer.AddEntry(entry)
```

- `AddEntry` never blocks: it returns `false` and drops the entry when the queue
  is full.
- A background goroutine encodes queued entries, keeps them sorted by
  `startedDateTime`, and rewrites the whole file at most once per flush
  interval. Each write goes to a temporary file that is
  renamed over the target, so the file on disk is always a valid archive.
- `Close` writes the remaining entries and stops the goroutine; `Discard` stops
  it without writing. Both are safe to call more than once.
- The flush and error handlers run on the writer goroutine.
