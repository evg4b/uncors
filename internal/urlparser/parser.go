// Package urlparser parses and validates URLs with support for wildcards.
// This is an internal package that provides custom URL parsing for the uncors project.
package urlparser

import (
	"errors"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/idna"
)

var (
	ErrEmptyHost   = errors.New("empty host")
	ErrEmptyPort   = errors.New("empty port")
	ErrInvalidHost = errors.New("invalid host")
	ErrEmptyURL    = errors.New("empty url")
)

// Parse parses raw URL string into the net/url URL struct.
// It uses the url.Parse() internally, but it slightly changes
// its behavior:
//  1. It forces the default scheme and port to http
//  2. It favors absolute paths over relative ones, thus "example.com"
//     is parsed into url.Host instead of url.Path.
//  3. It lowercase's the Host (not only the Scheme).
func Parse(rawURL string) (*url.URL, error) {
	return ParseWithDefaultScheme(rawURL, "")
}

// ParseWithDefaultScheme parses raw URL string with a custom default scheme.
// If the URL doesn't have a scheme, the provided scheme will be used.
// If scheme is empty, the URL will be parsed without a default scheme.
func ParseWithDefaultScheme(rawURL string, scheme string) (*url.URL, error) {
	rawURL = defaultScheme(rawURL, scheme)

	// Use net/url.Parse() now.
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	host, _, err := SplitHostPort(parsedURL)
	if err != nil {
		return nil, err
	}
	if err := checkHost(host); err != nil {
		return nil, err
	}

	parsedURL.Host = strings.ToLower(parsedURL.Host)
	parsedURL.Scheme = strings.ToLower(parsedURL.Scheme)

	return parsedURL, nil
}

func defaultScheme(rawURL, scheme string) string {
	// Force default http scheme, so net/url.Parse() doesn't
	// put both host and path into the (relative) path.
	if strings.Index(rawURL, "//") == 0 {
		// Leading double slashes (any scheme). Force http.
		if len(scheme) > 0 {
			return scheme + ":" + rawURL
		}

		return rawURL
	}
	if !strings.Contains(rawURL, "://") {
		if len(scheme) > 0 {
			// Missing scheme. Force http.
			return scheme + "://" + rawURL
		}

		return "//" + rawURL
	}

	return rawURL
}

var (
	domainRegexp = regexp.MustCompile(`^([a-zA-Z0-9-_*]{1,63}\.)*([a-zA-Z0-9-*]{1,63})$`)
	ipv4Regexp   = regexp.MustCompile(`^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$`)
	ipv6Regexp   = regexp.MustCompile(`^\[[a-fA-F0-9:]+]$`)
)

func checkHost(host string) error {
	if host == "" {
		return &url.Error{Op: "host", URL: host, Err: ErrEmptyHost}
	}

	host = strings.ToLower(host)
	if domainRegexp.MatchString(host) {
		return nil
	}

	if punycode, err := idna.ToASCII(host); err != nil {
		return err
	} else if domainRegexp.MatchString(punycode) {
		return nil
	}

	// IPv4 and IPv6.
	if ipv4Regexp.MatchString(host) || ipv6Regexp.MatchString(host) {
		return nil
	}

	return &url.Error{Op: "host", URL: host, Err: ErrInvalidHost}
}

// SplitHostPort splits network address of the form "host:port" into
// host and port. Unlike net.SplitHostPort(), it doesn't remove brackets
// from [IPv6] host, and it accepts net/url.URL struct instead of a string.
func SplitHostPort(parsedURL *url.URL) (string, string, error) {
	if parsedURL == nil {
		return "", "", &url.Error{Op: "host", URL: "", Err: ErrEmptyURL}
	}
	host := parsedURL.Host

	// Find last colon.
	index := strings.LastIndex(host, ":")
	if index == -1 {
		// No port found.
		return host, "", nil
	}

	// Return if the last colon is inside [IPv6] brackets.
	if strings.HasPrefix(host, "[") && strings.Contains(host[index:], "]") {
		// No port found.
		return host, "", nil
	}

	if index == len(host)-1 {
		return "", "", &url.Error{Op: "port", URL: parsedURL.String(), Err: ErrEmptyPort}
	}

	port := host[index+1:]
	host = host[:index]

	if _, err := strconv.Atoi(port); err != nil {
		return "", "", &url.Error{Op: "port", URL: parsedURL.String(), Err: err}
	}

	return host, port, nil
}
