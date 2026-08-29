// Package urlpattern parses the host patterns uncors maps traffic between.
//
// A mapping's `from:` and `to:` are not URLs: `http://{repo}.local.com:8080` is
// a pattern for matching them. Modelling it as a URL is what forced a fork of
// net/url to tolerate braces; parsing it as what it is takes a hundred lines and
// leaves real URLs to the standard library.
package urlpattern

import (
	"errors"
	"fmt"
	"net/netip"
	"strings"
)

var (
	ErrEmptyHost        = errors.New("host must not be empty")
	ErrControlCharacter = errors.New("host contains an invalid control character")
	ErrHostHasPath      = errors.New("host must not contain a path, query or fragment")
	ErrInvalidScheme    = errors.New("host has an invalid scheme")
	ErrInvalidHostname  = errors.New("host has an invalid hostname")
	ErrInvalidPort      = errors.New("host has an invalid port")
	ErrInvalidIPv6      = errors.New("host has an invalid IPv6 address")
	ErrInvalidBrackets  = errors.New("host has misplaced square brackets")
	ErrInvalidPlacehldr = errors.New("host has an invalid {placeholder}")
)

// Host is a mapping endpoint: an optional scheme, a hostname that may contain
// {placeholder} segments, and an optional port.
//
//nolint:recvcheck // UnmarshalText has to take a pointer; the rest reads better as a value
type Host struct {
	Scheme   string
	Hostname string
	Port     string
}

// Parse reads a host pattern such as "https://{tenant}.api.local:8443".
func Parse(raw string) (*Host, error) {
	if raw == "" {
		return nil, ErrEmptyHost
	}

	if containsControlByte(raw) {
		return nil, ErrControlCharacter
	}

	host := &Host{}
	rest := raw

	if index := strings.Index(rest, "://"); index >= 0 {
		scheme := rest[:index]

		err := validateScheme(scheme)
		if err != nil {
			return nil, err
		}

		host.Scheme = strings.ToLower(scheme)
		rest = rest[index+len("://"):]
	} else {
		rest = strings.TrimPrefix(rest, "//")
	}

	if strings.ContainsAny(rest, "/?#") {
		return nil, ErrHostHasPath
	}

	hostname, port, err := splitHostPort(rest)
	if err != nil {
		return nil, err
	}

	err = validatePort(port)
	if err != nil {
		return nil, err
	}

	err = validateHostname(hostname)
	if err != nil {
		return nil, err
	}

	host.Hostname = hostname
	host.Port = port

	return host, nil
}

// String reassembles the Host into its canonical form (scheme://hostname:port).
// The scheme and port parts are omitted when empty. IPv6 hostnames are wrapped
// in square brackets.
func (h Host) String() string {
	var builder strings.Builder

	if h.Scheme != "" {
		builder.WriteString(h.Scheme)
		builder.WriteString("://")
	}

	builder.WriteString(h.bracketedHostname())

	if h.Port != "" {
		builder.WriteByte(':')
		builder.WriteString(h.Port)
	}

	return builder.String()
}

// HostPort returns the "hostname:port" form without the scheme.
func (h Host) HostPort() string {
	if h.Port == "" {
		return h.bracketedHostname()
	}

	return h.bracketedHostname() + ":" + h.Port
}

// MarshalText encodes the host as its canonical string form.
func (h Host) MarshalText() ([]byte, error) {
	return []byte(h.String()), nil
}

// UnmarshalText decodes a host from a YAML or JSON string scalar.
func (h *Host) UnmarshalText(text []byte) error {
	parsed, err := Parse(string(text))
	if err != nil {
		return err
	}

	*h = *parsed

	return nil
}

func (h Host) bracketedHostname() string {
	if strings.Contains(h.Hostname, ":") {
		return "[" + h.Hostname + "]"
	}

	return h.Hostname
}

func splitHostPort(hostport string) (string, string, error) {
	if strings.HasPrefix(hostport, "[") {
		end := strings.Index(hostport, "]")
		if end < 0 {
			return "", "", ErrInvalidBrackets
		}

		hostname := hostport[1:end]

		address, err := netip.ParseAddr(hostname)
		if err != nil || !address.Is6() || address.Is4In6() {
			return "", "", fmt.Errorf("%w: %s", ErrInvalidIPv6, hostname)
		}

		rest := hostport[end+1:]
		if rest == "" {
			return hostname, "", nil
		}

		if !strings.HasPrefix(rest, ":") {
			return "", "", ErrInvalidBrackets
		}

		return hostname, rest[1:], nil
	}

	if strings.ContainsAny(hostport, "[]") {
		return "", "", ErrInvalidBrackets
	}

	index := strings.LastIndex(hostport, ":")
	if index < 0 {
		return hostport, "", nil
	}

	return hostport[:index], hostport[index+1:], nil
}

func validateScheme(scheme string) error {
	if scheme == "" {
		return ErrInvalidScheme
	}

	for index := range len(scheme) {
		char := scheme[index]

		switch {
		case isAlpha(char):
		case index > 0 && (isDigit(char) || char == '+' || char == '-' || char == '.'):
		default:
			return fmt.Errorf("%w: %s", ErrInvalidScheme, scheme)
		}
	}

	return nil
}

func validatePort(port string) error {
	for index := range len(port) {
		if !isDigit(port[index]) {
			return fmt.Errorf("%w: %s", ErrInvalidPort, port)
		}
	}

	return nil
}

func validateHostname(hostname string) error {
	if hostname == "" {
		return ErrInvalidHostname
	}

	// An IPv6 hostname arrives here already validated by splitHostPort.
	if strings.Contains(hostname, ":") {
		return nil
	}

	err := validatePlaceholders(hostname)
	if err != nil {
		return err
	}

	for index := range len(hostname) {
		char := hostname[index]

		switch {
		case isAlpha(char), isDigit(char):
		case char == '-', char == '.', char == '_', char == '{', char == '}':
		default:
			return fmt.Errorf("%w: %s", ErrInvalidHostname, hostname)
		}
	}

	return nil
}

// validatePlaceholders checks that every {name} in the hostname is well formed:
// opened once, closed, and not empty.
func validatePlaceholders(hostname string) error {
	open := false
	empty := true

	for index := range len(hostname) {
		switch hostname[index] {
		case '{':
			if open {
				return fmt.Errorf("%w: nested '{'", ErrInvalidPlacehldr)
			}

			open, empty = true, true
		case '}':
			if !open {
				return fmt.Errorf("%w: unmatched '}'", ErrInvalidPlacehldr)
			}

			if empty {
				return fmt.Errorf("%w: empty placeholder", ErrInvalidPlacehldr)
			}

			open = false
		default:
			empty = false
		}
	}

	if open {
		return fmt.Errorf("%w: unclosed '{'", ErrInvalidPlacehldr)
	}

	return nil
}

func containsControlByte(value string) bool {
	for index := range len(value) {
		if char := value[index]; char < ' ' || char == 0x7f {
			return true
		}
	}

	return false
}

func isAlpha(char byte) bool {
	return ('a' <= char && char <= 'z') || ('A' <= char && char <= 'Z')
}

func isDigit(char byte) bool {
	return '0' <= char && char <= '9'
}
