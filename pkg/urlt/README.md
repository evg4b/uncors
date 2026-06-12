# urlt — Modified URL Parser

`urlt` is a modified implementation of Go's `net/url` package with customized URL parsing behavior to support the specific requirements of the uncors project.

## Overview

This package provides URL parsing and query escaping functionality based on RFC 3986, with modifications to handle non-standard URL patterns and edge cases that are important for the uncors project.

## Features

- **URL Parsing**: Parse URLs using `Parse()` and `ParseRequestURI()`
- **Path Joining**: Safely join path segments with `JoinPath()`

## Main Functions

### `Parse(rawURL string) (*url.URL, error)`
Parses a raw URL string into a `url.URL` using uncors's opinionated, host-biased
rules:

- A bare authority such as `example.com` or `localhost:8080` is treated as a
  **host**, not a relative path.
- The host is validated and lowercased.
- `{name}` placeholders in the host (e.g. `http://{client}.demo.com`) are parsed
  directly — no escaping or masking.

```go
u, err := urlt.Parse("https://{tenant}.example.com/path?query=value")
if err != nil {
    log.Fatal(err)
}
```

### `ParseWithDefaultScheme(rawURL, scheme string) (*url.URL, error)`
Like `Parse`, but injects `scheme` when `rawURL` has none. An empty `scheme`
leaves the URL scheme-relative (`//host`).

### `SplitHostPort(u *url.URL) (host, port string, err error)`
Splits a parsed URL's host into host and port, keeping the brackets around an
`[IPv6]` literal and requiring a numeric port.

### `ToString(u *url.URL) string`
Serializes a URL exactly like `url.URL.String()`, except it formats the host
with this package's escaping, so `{name}` placeholders are preserved instead of
being percent-encoded (`http://{client}.demo.com` rather than
`http://%7Bclient%7D.demo.com`). It round-trips with `Parse`. Use it instead of
`url.URL.String()` whenever a URL may contain a placeholder host.

```go
u, _ := urlt.Parse("https://{tenant}.example.com/api")
urlt.ToString(u) // "https://{tenant}.example.com/api"
u.String()       // "https://%7Btenant%7D.example.com/api"
```

> The opinionated parser builds on the package's faithful `net/url`-style parser
> (kept internal as `parseRaw`), which is also what powers `ParseRequestURI`,
> `JoinPath` and the escaping helpers below.

### `ParseRequestURI(rawURL string) (*url.URL, error)`
Parses a URL as it would appear in an HTTP request. This is stricter than `Parse()` and:
- Only accepts absolute URIs or absolute paths
- Ignores fragment identifiers (as per HTTP spec)
- Returns an error for relative references

Example:
```go
u, err := urlt.ParseRequestURI("http://example.com:8080/path")
if err != nil {
    log.Fatal(err)
}
```

## Modifications from net/url

This implementation includes several modifications from Go's standard `net/url`:

1. **Custom handling of special characters**: Certain characters are handled differently to match uncors's URL parsing requirements
2. **Improved error handling**: Better error messages for invalid URLs
3. **Compatibility fixes**: Handles edge cases in URL normalization

## Return Types

The package returns standard Go `*url.URL` structures from the `net/url` package, so it's fully compatible with the standard library:

```go
type URL struct {
    Scheme      string
    Opaque      string    // encoded opaque data
    User        *Userinfo // username and password information
    Host        string    // host or host:port
    Path        string    // path (relative paths may omit leading slash)
    RawPath     string    // encoded path hint (see EscapedPath method)
    OmitHost    bool      // do not emit empty host
    RawQuery    string    // encoded query values, without '?'
    Fragment    string    // fragment for references, without '#'
    RawFragment string    // encoded fragment hint (see EscapedFragment method)
}
```

## Licensing

This package contains code derived from Go's standard library `net/url` package:
- Original code: Copyright 2009 The Go Authors, under BSD-style license
- Modifications: Copyright 2026 Evgeny Abramovich, under MIT license

See the `LICENSE` file in this directory for full licensing details.

## RFC Compliance

The package generally follows RFC 3986 (Uniform Resource Identifier) and RFC 6874 (IPv6 Zone Literals), except where modifications are needed for compatibility or specific use cases in the uncors project.
