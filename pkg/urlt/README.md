# urlt — Modified URL Parser

`urlt` is a modified implementation of Go's `net/url` package with customized URL parsing behavior to support the specific requirements of the uncors project.

## Overview

This package provides URL parsing and query escaping functionality based on RFC 3986, with modifications to handle non-standard URL patterns and edge cases that are important for the uncors project.

## Features

- **URL Parsing**: Parse URLs using `Parse()` and `ParseRequestURI()`
- **Query Encoding/Decoding**: `QueryEscape()` and `QueryUnescape()` for query parameters
- **Path Encoding/Decoding**: `PathEscape()` and `PathUnescape()` for URL paths
- **Query Parsing**: Parse URL query strings into `Values` (key-value pairs)
- **Path Joining**: Safely join path segments with `JoinPath()`

## Main Functions

### `Parse(rawURL string) (*url.URL, error)`
Parses a raw URL string into a `url.URL` structure. Unlike `ParseRequestURI()`, this function accepts:
- Absolute URIs (with scheme)
- Relative references
- Fragment identifiers

Example:
```go
u, err := urlt.Parse("https://example.com/path?query=value#fragment")
if err != nil {
    log.Fatal(err)
}
```

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

### Query and Path Escaping

```go
// Query parameter escaping
escaped := urlt.QueryEscape("hello world?")  // "hello+world%3F"
unescaped, err := urlt.QueryUnescape("hello+world%3F")

// Path segment escaping
pathEscaped := urlt.PathEscape("path/to/resource")
pathUnescaped, err := urlt.PathUnescape("path%2Fto%2Fresource")
```

### Query Parsing

```go
values, err := urlt.ParseQuery("name=John&age=30&hobby=reading")
if err != nil {
    log.Fatal(err)
}

// Access query values
fmt.Println(values.Get("name"))        // "John"
fmt.Println(values["hobby"])            // []string{"reading"}
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


Changes:
- Dropper suppoer of <>
