# har

Reads, builds, and writes [HAR 1.2](http://www.softwareishard.com/blog/har-12-spec/)
archives. It has no dependencies on the rest of uncors, so it can be used on its
own.

## API

### Types

`HAR`, `Log`, `Creator`, `Entry`, `Request`, `Response`, `Content`, `Timings`,
`NameValue`, `Cookie`, and `PostData` mirror the HAR 1.2 objects and marshal to
the field names of the spec.

### Archives

```go
archive := har.New(har.Creator{Name: "my-tool", Version: "1.0"}, entries...)

err := har.Encode(w, archive)   // indented JSON with a trailing newline
archive, err := har.Decode(r)
```

`har.Version` is the format version, `1.2`.

### Building entries from net/http

```go
request := har.NewRequest(req, requestBody)
response := har.NewResponse(resp, responseBody)
entry := har.NewEntry(start, elapsed, request, response)
```

- `NewRequest` takes the URL from `req.Host` and `req.RequestURI` for a server
  request (`https` when `req.TLS` is set), and from `req.URL` for a client
  request.
- `NewResponse` decodes the body by `Content-Encoding` (`gzip`, `deflate`, `br`,
  `zstd`) and stores it as text, or as base64 when it is not UTF-8 or cannot be
  decoded. `BodySize` is the size of the body as sent.
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
- A background goroutine encodes queued entries and rewrites the whole file at
  most once per flush interval. Each write goes to a temporary file that is
  renamed over the target, so the file on disk is always a valid archive.
- `Close` writes the remaining entries and stops the goroutine; `Discard` stops
  it without writing. Both are safe to call more than once.
- The flush and error handlers run on the writer goroutine.
