# urlt

A fork of the Go standard library [net/url](https://pkg.go.dev/net/url) package. Operates directly on `*net/url.URL` and exposes URL functionality as standalone functions rather than methods.

## API

### Parsing

```go
u, err := urlt.Parse("https://user:pass@example.com/path?q=1#frag")
u, err := urlt.ParseRequestURI("https://example.com/path")
```

### Stringification

```go
urlt.URL_String(u)           // reassemble URL into a string
urlt.URL_RequestURI(u)       // encoded path?query for HTTP requests
urlt.Redacted(u)             // like URL_String but masks the password
```

### Path and fragment

```go
urlt.URL_EscapedPath(u)      // percent-encoded form of u.Path
urlt.URL_EscapedFragment(u)  // percent-encoded form of u.Fragment
urlt.URL_JoinPath(u, elem...) // new URL with path elements appended
urlt.JoinPath(base, elem...)  // same, starting from a raw URL string

urlt.PathEscape(s)
urlt.PathUnescape(s)
```

### Query

```go
urlt.URL_Query(u)            // parses u.RawQuery, silently drops errors
urlt.ParseQuery(query)       // parses a query string, returns errors
urlt.Encode(values)          // encodes net/url.Values to query string

urlt.QueryEscape(s)
urlt.QueryUnescape(s)
```

### Host

```go
urlt.URL_Hostname(u)         // host without port
urlt.URL_Port(u)             // port without leading colon
```

### Reference resolution

```go
urlt.URL_ResolveReference(base, ref)  // resolve ref against base per RFC 3986
urlt.URL_Parse(base, ref)             // parse ref then resolve against base
```

### URLT type

`URLT` is a defined type over `net/url.URL` with custom binary marshaling. It serializes using `URL_String` and deserializes using `Parse`, instead of the standard library implementations.

```go
type MyConfig struct {
    Endpoint urlt.URLT
}
```

## Differences from net/url

**Functions instead of methods** — URL operations are standalone functions with a `URL_` prefix rather than methods on a `URL` type. Use `urlt.URL_String(u)` instead of `u.String()`, `urlt.URL_Query(u)` instead of `u.Query()`, etc.

**Works with `*net/url.URL` directly** — there is no custom `URL` struct. Parse returns a `*net/url.URL` and all functions accept one.

**No `Userinfo` or `Values` types** — uses `net/url.Userinfo` and `net/url.Values` from the standard library.

**No custom error types** — `EscapeError`, `InvalidHostError`, and `Error` are not re-exported; errors returned are the `net/url` equivalents.

**`URL.IsAbs()` removed** — check `u.Scheme != ""` directly.

**`URLT` type added** — `type URLT net/url.URL` with `MarshalBinary`/`UnmarshalBinary` that use `URL_String`/`Parse` instead of the standard library implementations.

**No `godebug`-gated behaviour or `//go:linkname` functions.**

---

Original code copyright 2009 The Go Authors. Licensed under the BSD-style license — see [LICENSE](LICENSE).
