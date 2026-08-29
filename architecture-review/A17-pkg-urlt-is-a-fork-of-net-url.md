# A17 — `pkg/urlt` is a 1 100-line fork of `net/url` maintained to allow `{}` in hostnames

**Severity:** Medium (maintenance burden / risk)
**Area:** URL handling

---

## 1. What is wrong with the current approach

`pkg/urlt` is a vendored, modified copy of the Go standard library's `net/url`:

```go
// pkg/urlt/url.go:1-15
// Copyright 2009 The Go Authors. All rights reserved.
// Portions Copyright 2026 Evgeny Abramovich. All rights reserved.
//
// Package urlt parses URLs and implements query escaping.
// It is a fork of the Go standard library's net/url package with modifications
// for use in uncors (no Userinfo struct, simplified API).
```

It is 1 476 lines across `url.go` (1 126), `gen_encoding_table.go` (234),
`encoding_table.go` (116) and `host.go` (114). The functional delta over the
standard library is small:

- allow `{name}` placeholders inside the host and reject them elsewhere
  ([`pkg/urlt/url.go:29-35`](../pkg/urlt/url.go#L29))
- a `Host` value type with `Scheme`/`Hostname`/`Port` and YAML text
  marshalling ([`pkg/urlt/host.go:8`](../pkg/urlt/host.go#L8))

Everything else — escaping tables, `parse`, `resolvePath`, `Values` encoding, the
whole surface — is standard-library code that must now be kept in sync by hand.

The fork also produces an awkward API. Because it keeps returning
`*net/url.URL`, the methods that would normally be on the value become
free functions with a non-Go naming convention:

```go
URL_String(u)        URL_Query(u)        URL_Hostname(u)   URL_Port(u)
URL_EscapedPath(u)   URL_RequestURI(u)   URL_ResolveReference(u, ref)
URL_JoinPath(u, ...) URL_Parse(u, ref)   URL_EscapedFragment(u)
```

Fifteen exported `URL_*` functions, all marked `//NOSONAR`, in a package that
`.golangci.yml` excludes from linting entirely
([`.golangci.yml`](../.golangci.yml)). So the largest single package in the
project is both unidiomatic and unlinted.

**The cost is paid on every request.** `URL_String` is called in the proxy hot
path ([`internal/handler/proxy/handler.go:79`](../internal/handler/proxy/handler.go#L79)),
in the URL replacer ([`internal/urlreplacer/factory.go:54`](../internal/urlreplacer/factory.go#L54)),
in HAR, in the cache key, and in the Lua request table — each call re-renders the
URL to a string so a regexp can be run over it
([A08](A08-proxy-handler-reimplements-reverse-proxy.md)).

**And the placeholder support is barely used by the parser.** The replacer does
not rely on `urlt`'s placeholder awareness for matching: `wildCardToRegexp`
extracts the host portion with plain string operations
([`internal/urlreplacer/helpers.go:56-88`](../internal/urlreplacer/helpers.go#L57)) and
`validateRawURL` **substitutes every `{key}` with `x` before parsing**
([`internal/urlreplacer/helpers.go:151`](../internal/urlreplacer/helpers.go#L151)):

```go
normalized := placeholderRegexp.ReplaceAllString(rawURL, "x")
...
parsed, err := urlt.Parse(normalized)
```

That is, the one place that validates a placeholder URL already works around the
placeholder rather than relying on the fork. The fork's placeholder support is
needed mainly by `ParseHost` ([`pkg/urlt/host.go:73`](../pkg/urlt/host.go#L73)), which
is called when decoding `from:`/`to:` from YAML.

## 2. Why it is an architectural problem

- **A standard-library fork is a permanent maintenance liability.** Security and
  correctness fixes to `net/url` (IPv6 zone handling, escaping edge cases, RFC
  updates) do not flow in automatically. The header already says "Portions
  Copyright 2026", meaning it has diverged; nothing tracks which upstream revision
  it forked from or whether it is current.
- **The problem being solved is a configuration-parsing problem, not a
  URL-parsing problem.** `http://{repo}.local.com:8080` is not a URL — it is a
  *pattern for matching URLs*. Modelling it as a URL forces the URL parser to
  accept something that is not one, which is why the escaping tables had to be
  regenerated and why braces need special-casing in paths and queries.
- **It inflates the project's apparent surface.** `pkg/urlt` is ~16 % of the
  non-test code and, being excluded from linting, is effectively unreviewed.

## 3. What the recommended approach is instead

**Separate the pattern type from the URL type.**

```go
// internal/config (or internal/hostpattern)
type HostPattern struct {           // parsed from "http://{repo}.local.com:8080"
	Scheme    string                // "", "http", "https"
	Segments  []segment             // literal | placeholder("repo")
	Port      string
}
func ParseHostPattern(s string) (HostPattern, error)   // hand-written, ~80 lines
func (p HostPattern) Regexp() *regexp.Regexp
func (p HostPattern) Render(vars map[string]string) string
```

`HostPattern` is what `from:` and `to:` decode into, and it is what
`urlreplacer` compiles. It never needs to be a `*url.URL`, so it never needs a
URL parser that tolerates braces.

**Everything that handles real URLs goes back to `net/url`.** Concretely:

- `Mapping.From` / `Mapping.To` become `HostPattern` instead of `urlt.Host`.
- The replacer transforms `*url.URL` structurally rather than by string regexp:
  match `u.Host` against the source pattern's regexp, render the target pattern
  with the captured vars, and copy `Path`/`RawQuery`/`Fragment` across untouched.
  (This is also what [A08](A08-proxy-handler-reimplements-reverse-proxy.md) needs.)
- Every `urlt.URL_String(u)` becomes `u.String()`; `urlt.URL_Query(u)` becomes
  `u.Query()`; and so on — the standard methods.
- `pkg/urlt` is deleted.

If some genuine `net/url` limitation remains after this (worth verifying case by
case), address it locally rather than by forking: most such needs are met by
manipulating `url.URL` fields directly rather than by re-parsing strings.

## 4. Why the proposed approach is better

- **~1 500 lines of forked standard-library code leave the repository**, along
  with the obligation to track upstream and the lint exclusion that hides it.
- **URL handling becomes correct-by-inheritance.** Go's `net/url` is
  battle-tested; a fork frozen at some 2026 revision is not.
- **The hot path loses a render→regexp→parse round trip per request**, replaced by
  a single regexp match on `u.Host` (a much shorter string) plus field copies.
- **The domain model becomes honest**: a mapping pattern is a pattern, a URL is a
  URL, and the type system says so. Today `Mapping.From` is a `urlt.Host` that may
  or may not contain placeholders, and every consumer has to know which.
- **`{}` handling gets simpler**, because it only has to work in one small
  hand-written parser instead of throughout an escaping/parsing engine.

## 5. Trade-offs and migration considerations

- **This is a substantial change** (the replacer, the config types, and ~15
  call sites of `URL_*`), and it should be sequenced with
  [A08](A08-proxy-handler-reimplements-reverse-proxy.md) since both touch the same
  code and the structural replacer is a prerequisite for `ReverseProxy.Rewrite`.
- **Behaviour differences must be pinned by tests first.** `pkg/urlt` has its own
  test suite ([`pkg/urlt/url_test.go`](../pkg/urlt/url_test.go),
  [`pkg/urlt/host_test.go`](../pkg/urlt/host_test.go)); before removal, extract the
  cases that describe *uncors* behaviour (placeholder hosts, scheme-less hosts,
  IPv6, default ports) into tests against the new `HostPattern`, and verify the
  integration suites in `tests/integration/domains` and `tests/integration/rewrite`
  still pass.
- **YAML marshalling must be preserved.** `urlt.Host` implements
  `MarshalText`/`UnmarshalText` ([`pkg/urlt/host.go:54`](../pkg/urlt/host.go#L54)) and
  the config round-trips through YAML in tests and in the integration harness;
  `HostPattern` needs the same.
- **Placeholder semantics should be tightened as part of the move.** Today
  `wildCardToRegexp` emits `(?P<key>.+)` ([`internal/urlreplacer/helpers.go:103`](../internal/urlreplacer/helpers.go#L103)),
  which is greedy and matches dots — contradicting the documentation's "excluding
  `.`" claim ([D10](D10-mapping-and-cors-behaviour-does-not-match-docs.md)). A
  hand-written pattern compiler should emit `[^.]+` per segment (or make the
  behaviour a documented choice).
- If removing the fork is judged too large for now, the minimum interim step is to
  **record the upstream Go revision it was forked from** in a header comment and
  **remove the lint exclusion**, so the divergence is at least trackable.

## 6. Code references

| What | Where |
| --- | --- |
| Fork header and scope | [`pkg/urlt/url.go:1`](../pkg/urlt/url.go#L1) |
| `URL_*` free functions | [`pkg/urlt/url.go:553`](../pkg/urlt/url.go#L553), [`:664`](../pkg/urlt/url.go#L664), [`:908`](../pkg/urlt/url.go#L908), [`:950`](../pkg/urlt/url.go#L950), [`:979`](../pkg/urlt/url.go#L979) |
| `Host` value type | [`pkg/urlt/host.go:8`](../pkg/urlt/host.go#L8) |
| Placeholder errors in the parser | [`pkg/urlt/url.go:29`](../pkg/urlt/url.go#L29) |
| Generated escaping table | [`pkg/urlt/gen_encoding_table.go`](../pkg/urlt/gen_encoding_table.go), [`pkg/urlt/encoding_table.go`](../pkg/urlt/encoding_table.go) |
| Replacer works around the parser | [`internal/urlreplacer/helpers.go:151`](../internal/urlreplacer/helpers.go#L151) |
| Greedy placeholder regexp | [`internal/urlreplacer/helpers.go:103`](../internal/urlreplacer/helpers.go#L103) |
| Hot-path `URL_String` | [`internal/handler/proxy/handler.go:79`](../internal/handler/proxy/handler.go#L79), [`internal/urlreplacer/factory.go:54`](../internal/urlreplacer/factory.go#L54) |
| Lint exclusion | [`.golangci.yml`](../.golangci.yml) |
