package urlt

import (
	"errors"
	"strings"
)

type Host struct {
	Scheme   string
	Hostname string
	Port     string
}

// String reassembles the Host into its canonical URL form
// (scheme://hostname:port). The scheme and port parts are omitted when empty.
// IPv6 hostnames are wrapped in square brackets.
func (h Host) String() string {
	var b strings.Builder

	if h.Scheme != "" {
		b.WriteString(h.Scheme)
		b.WriteString("://")
	}

	b.WriteString(h.bracketedHostname())

	if h.Port != "" {
		b.WriteByte(':')
		b.WriteString(h.Port)
	}

	return b.String()
}

// HostPort returns the "hostname:port" form without the scheme. When the port
// is empty only the hostname is returned. IPv6 hostnames are wrapped in
// square brackets.
func (h Host) HostPort() string {
	if h.Port == "" {
		return h.bracketedHostname()
	}

	return h.bracketedHostname() + ":" + h.Port
}

func (h Host) bracketedHostname() string {
	if strings.Contains(h.Hostname, ":") {
		return "[" + h.Hostname + "]"
	}

	return h.Hostname
}

// MarshalText implements encoding.TextMarshaler, allowing a Host to be encoded
// to YAML/JSON as its canonical string form.
func (h Host) MarshalText() ([]byte, error) {
	return []byte(h.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler, allowing a Host to be
// decoded from a YAML/JSON string scalar via ParseHost.
func (h *Host) UnmarshalText(text []byte) error {
	parsed, err := ParseHost(string(text))
	if err != nil {
		return err
	}

	*h = *parsed

	return nil
}

func ParseHost(rawURL string) (*Host, error) {
	var rest string
	var err error

	if rawURL == "" {
		return nil, errors.New("empty host")
	}

	if stringContainsCTLByte(rawURL) {
		return nil, errors.New("net/url: invalid control character in URL")
	}

	if !strings.Contains(rawURL, "//") {
		rawURL = "//" + rawURL
	}

	host := new(Host)

	if host.Scheme, rest, err = getScheme(rawURL); err != nil {
		return nil, err
	}

	host.Scheme = strings.ToLower(host.Scheme)

	rest = strings.TrimPrefix(rest, "//")

	if i := strings.Index(rest, "/"); i >= 0 {
		return nil, errors.New("host must not contain a path")
	}

	hostWithPort, err := parseHost(host.Scheme, rest)
	if err != nil {
		return nil, err
	}

	hostName, port := splitHostPort(hostWithPort)

	host.Hostname = hostName
	host.Port = port

	return host, nil
}
