// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// Portions of this code have been modified by Evgeny Abramovich.
// These modifications are provided under the MIT License.
// See the LICENSE file in this directory for details.

//go:generate go run gen_encoding_table.go

// Package urlt parses URLs and implements query escaping.
//
// See RFC 3986. This package generally follows RFC 3986, except where
// it deviates for compatibility reasons.
// RFC 6874 followed for IPv6 zone literals.
package urlt

// When sending changes, first  search old issues for history on decisions.
// Unit tests should also contain references to issue numbers with details.

import (
	"errors"
	"fmt"
	"net/netip"
	baseUrl "net/url"
	"path"
	"strings"
)

const upperhex = "0123456789ABCDEF"

const (
	schemeHTTP  = "http"
	schemeHTTPS = "https"
	opParse     = "parse"
)

var (
	errMissingProtocolScheme = errors.New("missing protocol scheme")
	errInvalidControlChar    = errors.New("net/url: invalid control character in URL")
	errEmptyURL              = errors.New("empty url")
	errInvalidURIForRequest  = errors.New("invalid URI for request")
	errFirstPathSegmentColon = errors.New("first path segment in URL cannot contain colon")
	errInvalidUserinfo       = errors.New("net/url: invalid userinfo")
	errInvalidIPLiteral      = errors.New("invalid IP-literal")
	errMissingCloseBracket   = errors.New("missing ']' in host")
)

func ishex(c byte) bool {
	return table[c]&hexChar != 0
}

// Precondition: ishex(c) is true.
func unhex(c byte) byte {
	return 9*(c>>6) + (c & 15)
}

// See the reference implementation in gen_encoding_table.go.
func shouldEscape(c byte, mode encoding) bool {
	return table[c]&mode == 0
}

// unescape unescapes a string; the mode specifies
// which section of the URL string is being unescaped.
//
//nolint:cyclop,gocognit
func unescape(s string, mode encoding) (string, error) { // NOSONAR
	// Count %, check that they're well-formed.
	n := 0
	hasPlus := false

	for i := 0; i < len(s); {
		switch s[i] {
		case '%':
			n++

			if i+2 >= len(s) || !ishex(s[i+1]) || !ishex(s[i+2]) {
				s = s[i:]
				if len(s) > 3 {
					s = s[:3]
				}

				return "", baseUrl.EscapeError(s)
			}
			// Per https://tools.ietf.org/html/rfc3986#page-21
			// in the host component %-encoding can only be used
			// for non-ASCII bytes.
			// But https://tools.ietf.org/html/rfc6874#section-2
			// introduces %25 being allowed to escape a percent sign
			// in IPv6 scoped-address literals. Yay.
			if mode == encodeHost && unhex(s[i+1]) < 8 && s[i:i+3] != "%25" {
				return "", baseUrl.EscapeError(s[i : i+3])
			}

			if mode == encodeZone {
				// RFC 6874 says basically "anything goes" for zone identifiers
				// and that even non-ASCII can be redundantly escaped,
				// but it seems prudent to restrict %-escaped bytes here to those
				// that are valid host name bytes in their unescaped form.
				// That is, you can use escaping in the zone identifier but not
				// to introduce bytes you couldn't just write directly.
				// But Windows puts spaces here! Yay.
				v := unhex(s[i+1])<<4 | unhex(s[i+2])
				if s[i:i+3] != "%25" && v != ' ' && shouldEscape(v, encodeHost) {
					return "", baseUrl.EscapeError(s[i : i+3])
				}
			}

			i += 3
		case '+':
			hasPlus = mode == encodeQueryComponent
			i++
		default:
			if (mode == encodeHost || mode == encodeZone) && s[i] < 0x80 && shouldEscape(s[i], mode) {
				return "", baseUrl.InvalidHostError(s[i : i+1])
			}

			i++
		}
	}

	if n == 0 && !hasPlus {
		return s, nil
	}

	var unescapedPlusSign byte

	switch mode { //nolint:exhaustive
	case encodeQueryComponent:
		unescapedPlusSign = ' '
	default:
		unescapedPlusSign = '+'
	}

	var t strings.Builder
	t.Grow(len(s) - 2*n)

	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '%':
			// In the loop above, we established that unhex's precondition is
			// fulfilled for both s[i+1] and s[i+2].
			t.WriteByte(unhex(s[i+1])<<4 | unhex(s[i+2]))
			i += 2
		case '+':
			t.WriteByte(unescapedPlusSign)
		default:
			t.WriteByte(s[i])
		}
	}

	return t.String(), nil
}

//nolint:cyclop
func escape(s string, mode encoding) string { // NOSONAR
	spaceCount, hexCount := 0, 0

	for _, c := range []byte(s) {
		if shouldEscape(c, mode) {
			if c == ' ' && mode == encodeQueryComponent {
				spaceCount++
			} else {
				hexCount++
			}
		}
	}

	if spaceCount == 0 && hexCount == 0 {
		return s
	}

	var (
		buf [64]byte
		t   []byte
	)

	required := len(s) + 2*hexCount
	if required <= len(buf) {
		t = buf[:required]
	} else {
		t = make([]byte, required)
	}

	if hexCount == 0 {
		copy(t, s)

		for i := range len(s) {
			if s[i] == ' ' {
				t[i] = '+'
			}
		}

		return string(t)
	}

	j := 0

	for _, c := range []byte(s) {
		switch {
		case c == ' ' && mode == encodeQueryComponent:
			t[j] = '+'
			j++
		case shouldEscape(c, mode):
			t[j] = '%'
			t[j+1] = upperhex[c>>4]
			t[j+2] = upperhex[c&15]
			j += 3
		default:
			t[j] = c
			j++
		}
	}

	return string(t)
}

// Maybe rawURL is of the form scheme:path.
// (Scheme must be [a-zA-Z][a-zA-Z0-9+.-]*)
// If so, return scheme, path; else return "", rawURL.
//
//nolint:cyclop
func getScheme(rawURL, defaultScheme string) (scheme, path string, err error) {
	for i := range len(rawURL) {
		c := rawURL[i]
		switch {
		case 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z':
		// do nothing
		case '0' <= c && c <= '9' || c == '+' || c == '-' || c == '.':
			if i == 0 {
				return defaultScheme, rawURL, nil
			}
		case c == ':':
			if i == 0 {
				return "", "", errMissingProtocolScheme
			}

			return rawURL[:i], rawURL[i+1:], nil
		default:
			// we have encountered an invalid character,
			// so there is no valid scheme
			return defaultScheme, rawURL, nil
		}
	}

	return defaultScheme, rawURL, nil
}

// parseRaw parses a raw url into a [URL] structure.
//
// The url may be relative (a path, without a host) or absolute
// (starting with a scheme). Trying to parse a hostname and path
// without a scheme is invalid but may not necessarily return an
// error, due to parsing ambiguities.
//
// parseRaw is the faithful net/url-style parser. The opinionated, host-biased
// [Parse]/[ParseWithDefaultScheme] in parser.go build on top of it.
func parseRaw(rawURL string) (*baseUrl.URL, error) {
	return parseRawWithDefaultScheme(rawURL, "")
}

func parseRawWithDefaultScheme(rawURL, defaultScheme string) (*baseUrl.URL, error) {
	// Cut off #frag
	u, frag, _ := strings.Cut(rawURL, "#")

	url, err := parse(u, defaultScheme, false)
	if err != nil {
		return nil, &baseUrl.Error{Op: opParse, URL: u, Err: err}
	}

	if frag == "" {
		return url, nil
	}

	if err = setFragment(url, frag); err != nil {
		return nil, &baseUrl.Error{Op: opParse, URL: rawURL, Err: err}
	}

	return url, nil
}

// ParseRequestURI parses a raw url into a [URL] structure. It assumes that
// url was received in an HTTP request, so the url is interpreted
// only as an absolute URI or an absolute path.
// The string url is assumed not to have a #fragment suffix.
// (Web browsers strip #fragment before sending the URL to a web server.)
func ParseRequestURI(rawURL string) (*baseUrl.URL, error) {
	return ParseRequestURIWithDefaultScheme(rawURL, "")
}

func ParseRequestURIWithDefaultScheme(rawURL, defaultScheme string) (*baseUrl.URL, error) {
	url, err := parse(rawURL, defaultScheme, true)
	if err != nil {
		return nil, &baseUrl.Error{Op: opParse, URL: rawURL, Err: err}
	}

	return url, nil
}

// parse parses a URL from a string in one of two contexts. If
// viaRequest is true, the URL is assumed to have arrived via an HTTP request,
// in which case only absolute URLs or path-absolute relative URLs are allowed.
// If viaRequest is false, all forms of relative URLs are allowed.
//
//nolint:cyclop,gocognit
func parse(rawURL, defaultScheme string, viaRequest bool) (*baseUrl.URL, error) { // NOSONAR
	var (
		rest string
		err  error
	)

	if stringContainsCTLByte(rawURL) {
		return nil, errInvalidControlChar
	}

	if rawURL == "" && viaRequest {
		return nil, errEmptyURL
	}

	url := new(baseUrl.URL)

	if rawURL == "*" {
		url.Path = "*"

		return url, nil
	}

	// Split off possible leading "http:", "mailto:", etc.
	// Cannot contain escaped characters.
	if url.Scheme, rest, err = getScheme(rawURL, defaultScheme); err != nil {
		return nil, err
	}

	url.Scheme = strings.ToLower(url.Scheme)

	if strings.HasSuffix(rest, "?") && strings.Count(rest, "?") == 1 {
		url.ForceQuery = true
		rest = rest[:len(rest)-1]
	} else {
		rest, url.RawQuery, _ = strings.Cut(rest, "?")
	}

	if !strings.HasPrefix(rest, "/") {
		if url.Scheme != "" {
			// We consider rootless paths per RFC 3986 as opaque.
			url.Opaque = rest

			return url, nil
		}

		if viaRequest {
			return nil, errInvalidURIForRequest
		}

		// Avoid confusion with malformed schemes, like cache_object:foo/bar.
		// See golang.org/issue/16822.
		//
		// RFC 3986, §3.3:
		// In addition, a URI reference (Section 4.1) may be a relative-path reference,
		// in which case the first path segment cannot contain a colon (":") character.
		if segment, _, _ := strings.Cut(rest, "/"); strings.Contains(segment, ":") {
			// First path segment has colon. Not allowed in relative URL.
			return nil, errFirstPathSegmentColon
		}
	}

	if (url.Scheme != "" || !viaRequest && !strings.HasPrefix(rest, "///")) && strings.HasPrefix(rest, "//") {
		var authority string

		authority, rest = rest[2:], ""
		if i := strings.Index(authority, "/"); i >= 0 {
			authority, rest = authority[:i], authority[i:]
		}

		url.User, url.Host, err = parseAuthority(url.Scheme, authority)
		if err != nil {
			return nil, err
		}
	} else if url.Scheme != "" && strings.HasPrefix(rest, "/") {
		// OmitHost is set to true when rawURL has an empty host (authority).
		// See golang.org/issue/46059.
		url.OmitHost = true
	}

	// Set Path and, optionally, RawPath.
	// RawPath is a hint of the encoding of Path. We don't want to set it if
	// the default escaping of Path is equivalent, to help make sure that people
	// don't rely on it in general.
	if err := setPath(url, rest); err != nil {
		return nil, err
	}

	return url, nil
}

func parseAuthority(scheme, authority string) (user *baseUrl.Userinfo, host string, err error) {
	i := strings.LastIndex(authority, "@")
	if i < 0 {
		host, err = parseHost(scheme, authority)
	} else {
		host, err = parseHost(scheme, authority[i+1:])
	}

	if err != nil {
		return nil, "", err
	}

	if i < 0 {
		return nil, host, nil
	}

	userinfo := authority[:i]
	if !validUserinfo(userinfo) {
		return nil, "", errInvalidUserinfo
	}

	if !strings.Contains(userinfo, ":") {
		if userinfo, err = unescape(userinfo, encodeUserPassword); err != nil {
			return nil, "", err
		}

		user = baseUrl.User(userinfo)
	} else {
		username, password, _ := strings.Cut(userinfo, ":")
		if username, err = unescape(username, encodeUserPassword); err != nil {
			return nil, "", err
		}

		if password, err = unescape(password, encodeUserPassword); err != nil {
			return nil, "", err
		}

		user = baseUrl.UserPassword(username, password)
	}

	return user, host, nil
}

// parseHost parses host as an authority without user
// information. That is, as host[:port].
//
//nolint:cyclop,gocognit
func parseHost(scheme, host string) (string, error) { // NOSONAR
	if openBracketIdx := strings.LastIndex(host, "["); openBracketIdx > 0 {
		return "", errInvalidIPLiteral
	} else if openBracketIdx == 0 {
		// Parse an IP-Literal in RFC 3986 and RFC 6874.
		// E.g., "[fe80::1]", "[fe80::1%25en0]", "[fe80::1]:80".
		closeBracketIdx := strings.LastIndex(host, "]")
		if closeBracketIdx < 0 {
			return "", errMissingCloseBracket
		}

		colonPort := host[closeBracketIdx+1:]
		if !validOptionalPort(colonPort) {
			return "", fmt.Errorf("invalid port %q after host", colonPort) //nolint:err113
		}

		unescapedColonPort, err := unescape(colonPort, encodeHost)
		if err != nil {
			return "", err
		}

		hostname := host[openBracketIdx+1 : closeBracketIdx]

		var unescapedHostname string
		// RFC 6874 defines that %25 (%-encoded percent) introduces
		// the zone identifier, and the zone identifier can use basically
		// any %-encoding it likes. That's different from the host, which
		// can only %-encode non-ASCII bytes.
		// We do impose some restrictions on the zone, to avoid stupidity
		// like newlines.
		zoneIdx := strings.Index(hostname, "%25")
		if zoneIdx >= 0 {
			hostPart, err := unescape(hostname[:zoneIdx], encodeHost)
			if err != nil {
				return "", err
			}

			zonePart, err := unescape(hostname[zoneIdx:], encodeZone)
			if err != nil {
				return "", err
			}

			unescapedHostname = hostPart + zonePart
		} else {
			var err error

			unescapedHostname, err = unescape(hostname, encodeHost)
			if err != nil {
				return "", err
			}
		}

		// Per RFC 3986, only a host identified by a valid
		// IPv6 address can be enclosed by square brackets.
		// This excludes any IPv4, but notably not IPv4-mapped addresses.
		addr, err := netip.ParseAddr(unescapedHostname)
		if err != nil {
			return "", fmt.Errorf("invalid host: %w", err) //nolint:err113
		}

		if addr.Is4() {
			return "", errInvalidIPLiteral
		}

		return "[" + unescapedHostname + "]" + unescapedColonPort, nil
	} else if i := strings.Index(host, ":"); i != -1 {
		lastColon := strings.LastIndex(host, ":")
		if lastColon != i {
			// RFC 3986 does not allow colons to appear in the host subcomponent.
			//
			// However, a number of databases including PostgreSQL and MongoDB
			// permit a comma-separated list of hosts (with optional ports) in the
			// host subcomponent.
			//
			// Since we historically permitted colons to appear in the host,
			// enforce strict colons only for http and https URLs.
			//
			// See https://go.dev/issue/75223 and https://go.dev/issue/78077.
			if scheme != schemeHTTP && scheme != schemeHTTPS {
				i = lastColon
			}
		}

		colonPort := host[i:]
		if !validOptionalPort(colonPort) {
			return "", fmt.Errorf("invalid port %q after host", colonPort) //nolint:err113
		}
	}

	var err error
	if host, err = unescape(host, encodeHost); err != nil {
		return "", err
	}

	return host, nil
}

// setPath sets the Path and RawPath fields of the URL based on the provided
// escaped path p. It maintains the invariant that RawPath is only specified
// when it differs from the default encoding of the path.
// For example:
// - setPath("/foo/bar")   will set Path="/foo/bar" and RawPath=""
// - setPath("/foo%2fbar") will set Path="/foo/bar" and RawPath="/foo%2fbar"
// setPath will return an error only if the provided path contains an invalid
// escaping.
//
// Do not remove or change the type signature.
// See go.dev/issue/67401.
func setPath(u *baseUrl.URL, p string) error {
	path, err := unescape(p, encodePath)
	if err != nil {
		return err
	}

	u.Path = path
	if p == escape(path, encodePath) {
		// Default encoding is fine.
		u.RawPath = ""
	} else {
		u.RawPath = p
	}

	return nil
}

// setFragment is like setPath but for Fragment/RawFragment.
func setFragment(u *baseUrl.URL, f string) error {
	frag, err := unescape(f, encodeFragment)
	if err != nil {
		return err
	}

	u.Fragment = frag
	if f == escape(frag, encodeFragment) {
		// Default encoding is fine.
		u.RawFragment = ""
	} else {
		u.RawFragment = f
	}

	return nil
}

// validOptionalPort reports whether port is either an empty string
// or matches /^:\d*$/.
func validOptionalPort(port string) bool {
	if port == "" {
		return true
	}

	if port[0] != ':' {
		return false
	}

	for _, b := range port[1:] {
		if b < '0' || b > '9' {
			return false
		}
	}

	return true
}

// resolvePath applies special path segments from refs and applies
// them to base, per RFC 3986.
//
//nolint:cyclop
func resolvePath(base, ref string) string { // NOSONAR
	var full string

	switch {
	case ref == "":
		full = base
	case ref[0] != '/':
		i := strings.LastIndex(base, "/")
		full = base[:i+1] + ref
	default:
		full = ref
	}

	if full == "" {
		return ""
	}

	var (
		elem string
		dst  strings.Builder
	)

	first := true
	remaining := full
	// We want to return a leading '/', so write it now.
	dst.WriteByte('/')

	found := true
	for found {
		elem, remaining, found = strings.Cut(remaining, "/")
		if elem == "." {
			first = false
			// drop
			continue
		}

		if elem == ".." {
			// Ignore the leading '/' we already wrote.
			str := dst.String()[1:]
			index := strings.LastIndexByte(str, '/')

			dst.Reset()
			dst.WriteByte('/')

			if index == -1 {
				first = true
			} else {
				dst.WriteString(str[:index])
			}
		} else {
			if !first {
				dst.WriteByte('/')
			}

			dst.WriteString(elem)

			first = false
		}
	}

	if elem == "." || elem == ".." {
		dst.WriteByte('/')
	}

	// We wrote an initial '/', but we don't want two.
	r := dst.String()
	if len(r) > 1 && r[1] == '/' {
		r = r[1:]
	}

	return r
}

func joinPath(u *baseUrl.URL, elem ...string) (*baseUrl.URL, error) {
	elem = append([]string{u.EscapedPath()}, elem...)

	var p string

	if !strings.HasPrefix(elem[0], "/") {
		// Return a relative path if u is relative,
		// but ensure that it contains no ../ elements.
		elem[0] = "/" + elem[0]
		p = path.Join(elem...)[1:]
	} else {
		p = path.Join(elem...)
	}
	// path.Join will remove any trailing slashes.
	// Preserve at least one.
	if strings.HasSuffix(elem[len(elem)-1], "/") && !strings.HasSuffix(p, "/") {
		p += "/"
	}

	url := *u
	err := setPath(&url, p)

	return &url, err
}

// validUserinfo reports whether s is a valid userinfo string per RFC 3986
// Section 3.2.1:
//
//	userinfo    = *( unreserved / pct-encoded / sub-delims / ":" )
//	unreserved  = ALPHA / DIGIT / "-" / "." / "_" / "~"
//	sub-delims  = "!" / "$" / "&" / "'" / "(" / ")"
//	              / "*" / "+" / "," / ";" / "="
//
// It doesn't validate pct-encoded. The caller does that via func unescape.
//
//nolint:cyclop
func validUserinfo(s string) bool {
	for _, r := range s {
		if 'A' <= r && r <= 'Z' {
			continue
		}

		if 'a' <= r && r <= 'z' {
			continue
		}

		if '0' <= r && r <= '9' {
			continue
		}

		switch r {
		case '-', '.', '_', ':', '~', '!', '$', '&', '\'',
			'(', ')', '*', '+', ',', ';', '=', '%':
			continue
		case '@':
			// `RFC 3986 section 3.2.1` does not allow '@' in userinfo.
			// It is a delimiter between userinfo and host.
			// However, URLs are diverse, and in some cases,
			// the userinfo may contain an '@' character,
			// for example, in "http://username:p@ssword@google.com",
			// the string "username:p@ssword" should be treated as valid userinfo.
			// Ref:
			//   https://go.dev/issue/3439
			//   https://go.dev/issue/22655
			continue
		default:
			return false
		}
	}

	return true
}

// stringContainsCTLByte reports whether s contains any ASCII control character.
func stringContainsCTLByte(s string) bool {
	for i := range len(s) {
		b := s[i]
		if b < ' ' || b == 0x7f {
			return true
		}
	}

	return false
}

// JoinPath returns a [URL] string with the provided path elements joined to
// the existing path of base and the resulting path cleaned of any ./ or ../ elements.
// Path elements must already be in escaped form, as produced by [PathEscape].
func JoinPath(base string, elem ...string) (result string, err error) {
	url, err := parseRaw(base)
	if err != nil {
		return
	}

	res, err := joinPath(url, elem...)
	if err != nil {
		return "", err
	}

	return res.String(), nil
}
