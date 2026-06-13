package urlt

// This file hosts the opinionated, host-biased URL parser that uncors uses
// (migrated from the former internal/urlparser package). It builds directly on
// the package's faithful net/url-style parser (parseRaw) - which already accepts
// "{placeholder}" hosts thanks to the encoding table - so placeholders are
// handled by direct parsing, with no escaping or masking tricks.

import (
	"errors"
	"net/url"
	"strconv"
	"strings"
)

var (
	ErrEmptyHost             = errors.New("empty host")
	ErrEmptyPort             = errors.New("empty port")
	ErrEmptyURL              = errors.New("empty url")
	ErrInvalidHostChar       = errors.New("invalid character in host")
	ErrEmptyPlaceholder      = errors.New("empty placeholder in host")
	ErrUnclosedPlaceholder   = errors.New("unclosed placeholder in host")
	ErrUnmatchedClosingBrace = errors.New("unmatched closing brace in host")
)

const (
	hostOperation = "host"
	portOperation = "port"
)

// Parse parses rawURL into a net/url URL.
//
// Unlike [parseRaw] it favors absolute hosts over relative paths (so "demo.com"
// becomes Host, not Path), lowercases the host, requires a non-empty host, and
// natively accepts "{name}" placeholders in the host (e.g.
// "http://{client}.demo.com").
func Parse(rawURL string) (*url.URL, error) {
	return ParseWithDefaultScheme(rawURL, "")
}

// ParseWithDefaultScheme is like [Parse] but, when rawURL has no scheme, applies
// the provided scheme. An empty scheme leaves the URL scheme-less ("//host").
func ParseWithDefaultScheme(rawURL, scheme string) (*url.URL, error) {
	parsedURL, err := parseRaw(applyDefaultScheme(rawURL, scheme))
	if err != nil {
		return nil, err
	}

	host, _, err := SplitHostPort(parsedURL)
	if err != nil {
		return nil, err
	}

	err = checkHost(host)
	if err != nil {
		return nil, err
	}

	parsedURL.Host = strings.ToLower(parsedURL.Host)
	parsedURL.Scheme = strings.ToLower(parsedURL.Scheme)

	return parsedURL, nil
}

// ToString serializes u the way net/url's URL.String does, except that it
// escapes the host with this package's own escape function. Because the
// package's encoding table leaves "{" and "}" unescaped in a host, ToString
// preserves "{name}" placeholders (e.g. "http://{client}.demo.com") that the
// standard library would percent-encode to "%7Bclient%7D". It round-trips with
// [Parse].
//
// It is a faithful copy of (*url.URL).String() so that every other component
// (path, query, fragment, userinfo) is formatted exactly like the standard
// library; only the host escaping differs.
func ToString(parsedURL *url.URL) string {
	var buf strings.Builder

	if parsedURL.Scheme != "" {
		buf.WriteString(parsedURL.Scheme)
		buf.WriteByte(':')
	}

	if parsedURL.Opaque != "" {
		buf.WriteString(parsedURL.Opaque)
		writeQueryFragment(&buf, parsedURL)

		return buf.String()
	}

	writeAuthority(&buf, parsedURL)

	path := parsedURL.EscapedPath()
	if path != "" && path[0] != '/' && parsedURL.Host != "" {
		buf.WriteByte('/')
	}

	if buf.Len() == 0 {
		// RFC 3986 §4.2: a first path segment containing ":" must be preceded by
		// "./" so it is not mistaken for a scheme.
		if segment, _, _ := strings.Cut(path, "/"); strings.Contains(segment, ":") {
			buf.WriteString("./")
		}
	}

	buf.WriteString(path)
	writeQueryFragment(&buf, parsedURL)

	return buf.String()
}

// writeAuthority writes the "//user@host" portion. The host is escaped with this
// package's escape (which keeps "{" and "}" intact) - the sole deviation from
// url.URL.String.
func writeAuthority(buf *strings.Builder, parsedURL *url.URL) {
	// Mirrors url.URL.String: skip the authority when there is nothing to write,
	// or when an empty host is deliberately omitted.
	if parsedURL.Host == "" && parsedURL.User == nil && (parsedURL.Scheme == "" || parsedURL.OmitHost) {
		return
	}

	if parsedURL.Host != "" || parsedURL.Path != "" || parsedURL.User != nil {
		buf.WriteString("//")
	}

	if parsedURL.User != nil {
		buf.WriteString(parsedURL.User.String())
		buf.WriteByte('@')
	}

	if parsedURL.Host != "" {
		buf.WriteString(escape(parsedURL.Host, encodeHost))
	}
}

func writeQueryFragment(buf *strings.Builder, parsedURL *url.URL) {
	if parsedURL.ForceQuery || parsedURL.RawQuery != "" {
		buf.WriteByte('?')
		buf.WriteString(parsedURL.RawQuery)
	}

	if parsedURL.Fragment != "" {
		buf.WriteByte('#')
		buf.WriteString(parsedURL.EscapedFragment())
	}
}

// applyDefaultScheme rewrites rawURL so that parseRaw treats the leading token as
// a host rather than a relative path, optionally injecting a default scheme.
func applyDefaultScheme(rawURL, scheme string) string {
	switch {
	case strings.HasPrefix(rawURL, "//"):
		// Scheme-relative URL. Prefix the scheme only if one is requested.
		if scheme != "" {
			return scheme + ":" + rawURL
		}

		return rawURL
	case !strings.Contains(rawURL, "://"):
		// No scheme at all. Force "//" so the host is not parsed as a path.
		if scheme != "" {
			return scheme + "://" + rawURL
		}

		return "//" + rawURL
	default:
		return rawURL
	}
}

// maxASCII is the largest ASCII code point.
const maxASCII = 127

// isValidHostRune reports whether r is allowed in a DNS name, IPv4 address,
// or uncors {placeholder} pattern.  Non-ASCII runes (IDN labels) are always
// accepted; the encoding table already rejects control characters.
//
//nolint:cyclop
func isValidHostRune(char rune) bool {
	return char > maxASCII ||
		(char >= 'a' && char <= 'z') ||
		(char >= 'A' && char <= 'Z') ||
		(char >= '0' && char <= '9') ||
		char == '-' || char == '.' || char == '_' ||
		char == '{' || char == '}'
}

// checkHost rejects an empty host and hosts whose characters are not valid in
// a DNS name, IPv4/IPv6 address, or uncors {placeholder} pattern.
// The encoding table already handles control characters and special symbols
// like "<" and ">"; this layer additionally rejects RFC 3986 sub-delimiters
// (e.g. "&", "!") that are legal per the spec but meaningless in real hostnames.
func checkHost(host string) error {
	if host == "" {
		return &url.Error{Op: hostOperation, URL: host, Err: ErrEmptyHost}
	}

	// IPv6 bracket notation is already fully validated by parseHost.
	if strings.HasPrefix(host, "[") {
		return nil
	}

	// Strip the port (if any) so we only inspect the name/IP portion.
	if i := strings.LastIndex(host, ":"); i > 0 {
		host = host[:i]
	}

	for _, char := range host {
		if !isValidHostRune(char) {
			return &url.Error{Op: hostOperation, URL: host, Err: ErrInvalidHostChar}
		}
	}

	return validatePlaceholders(host)
}

// validatePlaceholders checks that every "{name}" placeholder in host is well-formed:
// non-empty, properly opened with "{", and properly closed with "}".
// It performs a single byte scan with no allocations.
func validatePlaceholders(host string) error {
	inPlaceholder := false
	placeholderLen := 0

	for i := range len(host) {
		switch host[i] {
		case '{':
			if inPlaceholder {
				return &url.Error{Op: hostOperation, URL: host, Err: ErrUnclosedPlaceholder}
			}
			inPlaceholder = true
			placeholderLen = 0
		case '}':
			if !inPlaceholder {
				return &url.Error{Op: hostOperation, URL: host, Err: ErrUnmatchedClosingBrace}
			}
			if placeholderLen == 0 {
				return &url.Error{Op: hostOperation, URL: host, Err: ErrEmptyPlaceholder}
			}
			inPlaceholder = false
		default:
			if inPlaceholder {
				placeholderLen++
			}
		}
	}

	if inPlaceholder {
		return &url.Error{Op: hostOperation, URL: host, Err: ErrUnclosedPlaceholder}
	}

	return nil
}

// SplitHostPort splits a URL's host into host and port. Unlike net.SplitHostPort
// it keeps the brackets around an [IPv6] host and takes a *url.URL.
func SplitHostPort(parsedURL *url.URL) (string, string, error) {
	if parsedURL == nil {
		return "", "", &url.Error{Op: hostOperation, URL: "", Err: ErrEmptyURL}
	}

	host := parsedURL.Host

	index := strings.LastIndex(host, ":")
	if index == -1 {
		return host, "", nil
	}

	// A trailing colon inside [IPv6] brackets is part of the address, not a port.
	if strings.HasPrefix(host, "[") && strings.Contains(host[index:], "]") {
		return host, "", nil
	}

	if index == len(host)-1 {
		return "", "", &url.Error{Op: portOperation, URL: parsedURL.String(), Err: ErrEmptyPort}
	}

	port := host[index+1:]

	_, err := strconv.Atoi(port)
	if err != nil {
		return "", "", &url.Error{Op: portOperation, URL: parsedURL.String(), Err: err}
	}

	return host[:index], port, nil
}
